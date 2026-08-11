/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package core is the organisation and the people in it.
 *
 * It is the module every other module assumes. Odoo calls this `base` and never
 * lets you uninstall it, for the same reason: an ERP without an answer to "what
 * is this organisation and who works here" is a set of screens that cannot
 * name their own subject. A document has to print a registration number, an
 * approval has to name a department, a deadline has to be counted in some
 * timezone — and until now none of those had anywhere to come from.
 *
 * The shape follows Odoo's, because the distinctions it draws are real:
 *
 *	res.company   → tenants + tenant_profiles   what the organisation is
 *	res.users     → users                       who the person is, anywhere
 *	hr.employee   → memberships                 who they are *here*
 *	hr.department → departments                 how the organisation is arranged
 *
 * The middle line is the one worth keeping straight. A language preference
 * belongs to a person and follows them between organisations; a job title does
 * not. The same person can be a director in one tenant and a clerk in another,
 * so the title lives on the membership and the language lives on the user.
 *
 * What this module does not do is authorisation. Roles and permissions already
 * exist and are already good (rbac + Settings → Access control); this names the
 * people those roles are handed to.
 */

package core

import (
	"net/http"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appregistry"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ID is the catalogue identifier. It is referenced by the platform, which
// installs this app for every tenant and refuses to disable it.
const ID = "io.gerege.nexus.core"

type Module struct {
	db    *pgxpool.Pool
	perms rbac.PermissionStore
}

func New(db *pgxpool.Pool) *Module {
	m := &Module{db: db, perms: rbac.NewSQLPermissionStore(db)}
	appregistry.Register(m)
	return m
}

func (m *Module) ID() string      { return ID }
func (m *Module) Name() string    { return "Organisation & People" }
func (m *Module) Version() string { return "1.0.0" }

func (m *Module) Dependencies() []internal.Dependency { return nil }

// Permissions are deliberately two, not six.
//
// Reading who works here is something any member of the organisation may do —
// a directory nobody can open is not a directory. Changing the organisation's
// legal identity, its structure, or who belongs to it is administrative, and it
// is one decision rather than three: anybody who can add a person to a
// department can already put them anywhere in it.
func (m *Module) Permissions() []internal.PermissionDefinition {
	return []internal.PermissionDefinition{
		{Code: "core.read", Name: "Read Organisation", Description: "View the organisation profile, its departments and its people"},
		{Code: "core.manage", Name: "Manage Organisation", Description: "Edit the organisation profile, its departments and its people"},
	}
}

// Menus are the three screens this app is. No parent is named: the platform
// hangs an app's own menus under its Modules group, and a ParentID set here
// would be overwritten — which reads as a decision that was silently ignored.
func (m *Module) Menus() []internal.MenuDefinition {
	return []internal.MenuDefinition{
		{
			ID: "core_organisation", Label: "Organisation",
			Path: "/organisation", Icon: "building-2", Order: 5,
			Labels: map[string]string{
				"mn": "Байгууллага", "ar": "المؤسسة", "zh": "组织",
				"fr": "Organisation", "ru": "Организация", "es": "Organización",
			},
		},
		{
			ID: "core_people", Label: "People",
			Path: "/organisation/people", Icon: "users", Order: 6,
			Labels: map[string]string{
				"mn": "Ажилтнууд", "ar": "الأشخاص", "zh": "人员",
				"fr": "Personnes", "ru": "Сотрудники", "es": "Personas",
			},
		},
		{
			ID: "core_departments", Label: "Departments",
			Path: "/organisation/departments", Icon: "network", Order: 7,
			Labels: map[string]string{
				"mn": "Хэлтэс, нэгж", "ar": "الأقسام", "zh": "部门",
				"fr": "Départements", "ru": "Подразделения", "es": "Departamentos",
			},
		},
	}
}

func (m *Module) RegisterRoutes(r chi.Router, tenantAuthMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/core", func(cr chi.Router) {
		cr.Use(tenantAuthMiddleware)
		read := rbac.RequirePermission(m.perms, "core.read")
		manage := rbac.RequirePermission(m.perms, "core.manage")

		// The organisation itself.
		cr.With(read).Get("/organisation", m.handleGetOrganisation)
		cr.With(manage).Put("/organisation", m.handleUpdateOrganisation)

		// How it is arranged.
		cr.With(read).Get("/departments", m.handleListDepartments)
		cr.With(manage).Post("/departments", m.handleCreateDepartment)
		cr.With(manage).Put("/departments/{id}", m.handleUpdateDepartment)
		cr.With(manage).Delete("/departments/{id}", m.handleArchiveDepartment)

		// Who is in it.
		cr.With(read).Get("/people", m.handleListPeople)
		cr.With(manage).Put("/people/{id}", m.handleUpdatePerson)
		cr.With(manage).Post("/people/{id}/deactivate", m.handleDeactivatePerson)
		cr.With(manage).Post("/people/{id}/reactivate", m.handleReactivatePerson)

		// What the signed-in person prefers, wherever they are. No permission:
		// these are the caller's own settings, and a person who cannot read
		// their own language preference has nothing to be protected from.
		cr.Get("/me/preferences", m.handleGetPreferences)
		cr.Put("/me/preferences", m.handleUpdatePreferences)
	})
}
