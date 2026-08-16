/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Package egov is this platform's front door to the Mongolian state's systems.
 *
 * The pieces existed already and were scattered: the ХУР registry lookups were
 * two handlers in the platform's own route table, sitting between the AI
 * endpoints and the integrations manager; whether eID, ДАН and ХУР were even
 * configured was knowable only from the deployment's environment; and what had
 * been looked up was in the audit log with nothing pointing at it. Between them
 * they are a product surface — "how this organisation is connected to the
 * state, and what it has asked the state" — and it had no name and no screen.
 *
 * What is here and what is deliberately not:
 *
 *	here     the registry lookups, the state of the three rails, and the
 *	         history of what this organisation looked up;
 *	platform the sign-in flows themselves (eID, ДАН, identity binding). They
 *	         run before anybody is signed in, so they cannot sit behind a gate
 *	         that asks which apps a tenant installed;
 *	platform a person's own list of linked identities and the button that
 *	         unlinks one. That is the account holder's, not their employer's,
 *	         and putting it behind an app an administrator can remove would
 *	         mean an administrator could take away somebody's ability to
 *	         detach their own national identity — see profile_handlers.go,
 *	         which has refused to be an app for the same reason since before
 *	         this module existed. This module links to it rather than owning
 *	         it;
 *	platform the low-level clients (platform/gerege, platform/eid,
 *	         platform/dan). They are infrastructure the platform itself signs
 *	         people in with; this module is their app-facing surface.
 */

package egov

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/gerege"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/staterail"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
)

// ID is the catalogue identifier.
const ID = "io.gerege.nexus.egov"

// Registry is what this module offers other modules in this binary.
//
// It is an interface and not the module type because the point of it is that
// nothing has to depend on this app being installed: a consumer holds a
// Registry or holds nil, and the nil case is a feature it does not offer rather
// than an error it reports. Contacts is the first caller-in-waiting — its
// address book fills itself in from the citizen registry — and it must keep
// working on a deployment that has no state integration at all.
type Registry interface {
	Citizen(ctx context.Context, regNumber string) (*gerege.CitizenInfo, error)
	Company(ctx context.Context, companyReg string) (*gerege.CompanyInfo, error)
}

// Rail and Rails moved to internal/platform/staterail: the platform builds the
// value and the platform is not allowed to import an app to name its type. See
// that package, and internal/apps/boundaries_test.go for the rule.

type Module struct {
	db    nexus.DB
	xyp   *gerege.GeregeService
	rails staterail.Rails
	perms nexus.PermissionStore
}

func New(p nexus.Platform, xyp *gerege.GeregeService, rails staterail.Rails) *Module {
	m := &Module{db: p.DB(), xyp: xyp, rails: rails, perms: p.Permissions()}
	nexus.Register(m)
	return m
}

// Compile-time proof that the module satisfies what it publishes.
var _ Registry = (*Module)(nil)

func (m *Module) Citizen(ctx context.Context, regNumber string) (*gerege.CitizenInfo, error) {
	return m.xyp.GetCitizenInfo(ctx, regNumber)
}

func (m *Module) Company(ctx context.Context, companyReg string) (*gerege.CompanyInfo, error) {
	return m.xyp.GetCompanyInfo(ctx, companyReg)
}

func (m *Module) ID() string { return ID }

// MenuPermission and RoutePermissionPrefix are this module's half of
// nexus.AccessPolicy — what the platform used to hold in a switch keyed by
// app ID, stated here so it survives the module moving to another repository.
// Route gating stays with the module: a citizen-registry lookup is a GET,
// and it must not be a `.read` that every member holds.
func (m *Module) MenuPermission() string        { return "egov.read" }
func (m *Module) RoutePermissionPrefix() string { return "" }
func (m *Module) Name() string                  { return "e-Government Link" }
func (m *Module) Version() string               { return "1.0.0" }

// Dependencies are none, in the direction that matters.
//
// Contacts reads the citizen registry through this module and does not depend
// on it: an address book that refuses to open because the state integration is
// missing would be a worse product than one that cannot pre-fill a form.
func (m *Module) Dependencies() []nexus.Dependency { return nil }

// Permissions: one to see the screens, two to ask the state a question.
//
// The two lookups are AdminOnly, which is the whole reason that field exists.
// They read another human being's record out of the national registry from
// nothing but their registration number, and the installer's default rule —
// anything ending `.read` goes to every member of the organisation — would
// have handed that to the whole staff on install. Before this module they were
// `xyp.citizen.read` and `xyp.company.read`, granted to the administrator role
// alone by migration 00024; this keeps that and says why in code rather than
// in a migration nobody re-reads.
func (m *Module) Permissions() []nexus.PermissionDefinition {
	return []nexus.PermissionDefinition{
		{Code: "egov.read", Name: "Read e-Government Link",
			Description: "See how this organisation is connected to the state's systems, and what it has looked up"},
		{Code: "egov.citizen.read", Name: "Query the citizen registry", AdminOnly: true,
			Description: "Look up authoritative citizen data through XYP"},
		{Code: "egov.company.read", Name: "Query the company registry", AdminOnly: true,
			Description: "Look up authoritative legal-entity data through XYP"},
	}
}

func (m *Module) Menus() []nexus.MenuDefinition {
	return []nexus.MenuDefinition{
		{ID: "egov_lookups", Label: "Registry lookups", Path: "/egov", Icon: "landmark", Order: 10,
			Labels: map[string]string{"mn": "Лавлагаа", "ar": "استعلامات السجل", "zh": "登记查询",
				"fr": "Consultations du registre", "ru": "Справки из реестра", "es": "Consultas al registro"}},
		{ID: "egov_connections", Label: "Connections", Path: "/egov/connections", Icon: "share-2", Order: 20,
			Labels: map[string]string{"mn": "Холболтууд", "ar": "الاتصالات", "zh": "连接",
				"fr": "Connexions", "ru": "Подключения", "es": "Conexiones"}},
		{ID: "egov_history", Label: "Lookup history", Path: "/egov/history", Icon: "scroll-text", Order: 30,
			Labels: map[string]string{"mn": "Түүх", "ar": "سجل الاستعلامات", "zh": "查询历史",
				"fr": "Historique", "ru": "История запросов", "es": "Historial"}},
	}
}

// RegisterRoutes mounts the app, and keeps the two pre-move URLs answering.
//
// The lookups were platform routes — /api/v1/xyp/* in the authenticated group —
// so the aliases move them behind this app's gate as well as renaming them. A
// tenant that removes this app loses the old URL too, which is the point:
// "uninstalled" has to mean the same thing at both addresses.
func (m *Module) RegisterRoutes(r chi.Router, tenantAuthMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/egov", func(er chi.Router) {
		er.Use(tenantAuthMiddleware)

		read := nexus.RequirePermission(m.perms, "egov.read")
		er.With(read).Get("/connections", m.handleConnections)
		er.With(read).Get("/history", m.handleHistory)

		er.With(nexus.RequirePermission(m.perms, "egov.citizen.read")).Post("/citizen", m.handleCitizen)
		er.With(nexus.RequirePermission(m.perms, "egov.company.read")).Post("/company", m.handleCompany)
	})

	// DEPRECATED: remove in vNEXT — the addresses these two lookups had while
	// they were platform routes.
	r.Route("/api/v1/xyp", func(xr chi.Router) {
		xr.Use(tenantAuthMiddleware)
		xr.With(nexus.RequirePermission(m.perms, "egov.citizen.read")).Post("/citizen", m.handleCitizen)
		xr.With(nexus.RequirePermission(m.perms, "egov.company.read")).Post("/company", m.handleCompany)
	})
}

func (m *Module) handleCitizen(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	var req struct {
		RegNumber string `json:"reg_number"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<13)).Decode(&req); err != nil || req.RegNumber == "" {
		nexus.Error(w, http.StatusBadRequest, "invalid registration number")
		return
	}

	info, err := m.Citizen(r.Context(), req.RegNumber)
	if err != nil {
		nexus.Error(w, http.StatusBadRequest, "XYP citizen query failed: "+err.Error())
		return
	}

	claims, _ := nexus.UserFromContext(r.Context())
	nexus.Audit(r.Context(), tenantID, claims.UserID, "egov.citizen_queried", "egov",
		map[string]any{"reg_number": req.RegNumber})

	nexus.JSON(w, http.StatusOK, info)
}

func (m *Module) handleCompany(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	var req struct {
		CompanyReg string `json:"company_reg"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<13)).Decode(&req); err != nil || req.CompanyReg == "" {
		nexus.Error(w, http.StatusBadRequest, "invalid company registration number")
		return
	}

	info, err := m.Company(r.Context(), req.CompanyReg)
	if err != nil {
		nexus.Error(w, http.StatusBadRequest, "XYP company query failed: "+err.Error())
		return
	}

	claims, _ := nexus.UserFromContext(r.Context())
	nexus.Audit(r.Context(), tenantID, claims.UserID, "egov.company_queried", "egov",
		map[string]any{"company_reg": req.CompanyReg})

	nexus.JSON(w, http.StatusOK, info)
}

// handleConnections answers with the three rails and where a person manages
// their own identity.
//
// The link out is part of the answer rather than something the client knows:
// this screen is the obvious place to look for "my eID", the control for it is
// deliberately not here, and a screen that mentions the thing without saying
// where it is would only send people looking through Settings.
func (m *Module) handleConnections(w http.ResponseWriter, r *http.Request) {
	if _, ok := nexus.RequireTenant(w, r); !ok {
		return
	}
	rails := []staterail.Rail{}
	if m.rails != nil {
		rails = m.rails()
	}
	nexus.JSON(w, http.StatusOK, map[string]any{
		"rails": rails,
		// Where the person's own linked identities live. See the package
		// comment for why they are not here.
		"identities_path": "/profile",
	})
}

// historyEntry is one thing this organisation asked the state.
type historyEntry struct {
	Action    string         `json:"action"`
	UserID    string         `json:"user_id"`
	Details   map[string]any `json:"details"`
	CreatedAt time.Time      `json:"created_at"`
}

// handleHistory reads the audit trail rather than a table of its own.
//
// The lookups already write an audit event, and a second record of the same act
// would be a second thing to keep in step with the first. The audit table is
// also the one place a deletion is not expected: an organisation should not be
// able to tidy away the record of whose registry data it read.
func (m *Module) handleHistory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	rows, err := m.db.Query(r.Context(),
		`SELECT action, COALESCE(user_id, ''), details, created_at
		   FROM audit_events
		  WHERE tenant_id = $1
		    AND (action LIKE 'egov.%' OR action LIKE 'xyp.%')
		  ORDER BY created_at DESC
		  LIMIT 200`, tenantID)
	if err != nil {
		slog.Error("egov: could not read the lookup history", "error", err, "tenant_id", tenantID)
		nexus.Error(w, http.StatusInternalServerError, "could not load the history")
		return
	}
	defer rows.Close()

	entries := make([]historyEntry, 0, 32)
	for rows.Next() {
		var e historyEntry
		var raw []byte
		if err := rows.Scan(&e.Action, &e.UserID, &raw, &e.CreatedAt); err != nil {
			continue
		}
		_ = json.Unmarshal(raw, &e.Details)
		entries = append(entries, e)
	}
	// `xyp.%` is in the query on purpose: events written before the rename are
	// the same acts under their old name, and a history that started empty on
	// the day this module shipped would look like a history that had been
	// cleared.
	nexus.JSON(w, http.StatusOK, entries)
}
