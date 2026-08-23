/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// Package staffpin is the short secret a person types to take over a shared
// till or kiosk.
//
// It was the platform's until 2026-08-23: two routes in server.go, a table, a
// lockout and a regular expression, all of it in a product nobody runs unless
// they run a shop. A queue kiosk in a hospital has no shift changes and a
// back-office deployment has no till, and both carried the whole of it.
//
// What could not move is the sign-in. `POST /api/v1/devices/staff/pin` still
// belongs to the platform, because what it produces is a platform session and
// minting one is not something an app may do — the SDK deliberately offers no
// way. So the route stayed and the *credential* left: this app implements
// nexus.StaffCredential, the platform's device route asks whatever implements
// it, and a deployment carrying no such app answers that it authenticates
// nobody on a device rather than that the PIN was wrong.
//
// The table stayed in db/migrations for now, like the assistant's two. Moving a
// schema is a data migration on every deployment that has one, and this app has
// not left the repository yet — the day it does, staff_pin_credentials goes
// with it through nexus.Migrations.
package staffpin

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// ID is the app id, the same one the catalogue and the store carry.
const ID = "io.gerege.nexus.staff_pin"

// PermissionManage is the only permission this app declares.
//
// Setting somebody else's PIN is setting how they sign in, so it is
// administrative by nature: AdminOnly withholds it from the default manager and
// user roles rather than letting the `.manage` suffix hand it to managers.
const PermissionManage = "staff_pin.manage"

// A PIN is four to twelve digits.
//
// Digits because it is typed on a till's number pad with a queue behind it; a
// floor rather than a length because six is a shop's business and four is a
// kiosk's. The ceiling exists so that nothing longer than a PIN is stored as
// one — a password typed here would be hashed into the wrong table.
var validPIN = regexp.MustCompile(`^[0-9]{4,12}$`)

// InstalledApps answers which apps an organisation has.
//
// An alias for the SDK's since 2026-08-23. It was a named type of this app's
// own, declared here because the platform published the capability under
// internal/apps' name — which a module outside this repository cannot ask for.
// The contract is exported now and this is one name for one thing again.
type InstalledApps = nexus.InstalledApps

// Module is the app.
type Module struct {
	db        nexus.DB
	perms     nexus.PermissionStore
	installed InstalledApps
}

// New builds the module, registers it, and publishes the credential.
//
// installed is the platform's own per-tenant gate, handed in. It is needed
// because the sign-in route this app answers for cannot carry the app gate: the
// request arrives with a device token and no session, and the app gate is built
// on a session. So the check moves inside Verify, where the tenant is known —
// an organisation that has not installed this app authenticates nobody through
// it.
func New(p nexus.Platform, installed InstalledApps) *Module {
	m := &Module{db: p.DB(), perms: p.Permissions(), installed: installed}
	nexus.Register(m)
	nexus.Provide[nexus.StaffCredential](m)
	return m
}

func (m *Module) ID() string      { return ID }
func (m *Module) Name() string    { return "Staff PIN" }
func (m *Module) Version() string { return "1.0.0" }

// Dependencies is empty: a till is enrolled by the platform, not by an app.
func (m *Module) Dependencies() []nexus.Dependency { return nil }

func (m *Module) Permissions() []nexus.PermissionDefinition {
	return []nexus.PermissionDefinition{
		{Code: PermissionManage, Name: "Manage Staff PINs",
			Description: "Set the PIN a member types to take over a shared till or kiosk",
			AdminOnly:   true},
	}
}

// Menus is empty: setting a PIN is done from the member's row in Access
// control, which is the platform's own screen.
func (m *Module) Menus() []nexus.MenuDefinition { return nil }

// RegisterRoutes mounts the one administrative route.
//
// The sign-in route is not here and cannot be: it is answered by the platform,
// which asks this app through nexus.StaffCredential. See the package comment.
func (m *Module) RegisterRoutes(r chi.Router, gateMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/admin/devices", func(ar chi.Router) {
		ar.Use(gateMiddleware)
		ar.Use(nexus.RequirePermission(m.perms, PermissionManage))

		ar.Put("/staff-pin", m.handleSetPIN)
	})
}

// handleSetPIN records a member's PIN, replacing whatever was there.
//
// Replacing rather than adding: a person has one PIN per membership, and the
// upsert clears the failed attempts and the lockout with it, because being
// given a new PIN is the administrator's answer to having been locked out of
// the old one.
func (m *Module) handleSetPIN(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "sign in first")
		return
	}
	var req struct {
		MembershipID string `json:"membership_id"`
		PIN          string `json:"pin"`
	}
	if httpx.DecodeLimited(r, &req, 8<<10) != nil || !validPIN.MatchString(req.PIN) {
		httpx.Error(w, http.StatusBadRequest, "PIN must contain 4-12 digits")
		return
	}
	hash, err := auth.HashPassword(req.PIN)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to protect PIN")
		return
	}
	// The membership is matched against the caller's own tenant in the SELECT,
	// so an id from another organisation writes nothing and reads as missing.
	result, err := m.db.Exec(r.Context(),
		`INSERT INTO staff_pin_credentials(membership_id,tenant_id,pin_hash)
		 SELECT id,tenant_id,$3 FROM memberships WHERE id=$1 AND tenant_id=$2
		 ON CONFLICT(membership_id) DO UPDATE
		    SET pin_hash=EXCLUDED.pin_hash,active=true,failed_attempts=0,locked_until=NULL,updated_at=NOW()`,
		req.MembershipID, claims.TenantID, hash)
	if err != nil || result.RowsAffected() != 1 {
		httpx.Error(w, http.StatusNotFound, "membership not found")
		return
	}
	audit.Record(r.Context(), claims.TenantID, claims.UserID, "staff.pin_changed", "membership",
		map[string]any{"membership_id": req.MembershipID})
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// Verify is nexus.StaffCredential: who typed this secret on this tenant's till.
//
// Every active credential of the organisation is read and compared in turn.
// That is deliberate and is the same shape the platform used before this app
// existed: a till knows the PIN and not who is typing it, so there is no row to
// look up by. The cost is bounded by the organisation's staff, and the hash
// comparison is the expensive part either way.
func (m *Module) Verify(ctx context.Context, tenantID, secret string) (nexus.StaffIdentity, error) {
	if !validPIN.MatchString(secret) {
		return nexus.StaffIdentity{}, nexus.ErrStaffCredentialRejected
	}
	// The per-tenant gate the app gate cannot apply here — see New.
	installed, err := m.installed(ctx, tenantID)
	if err != nil {
		return nexus.StaffIdentity{}, err
	}
	if !installed[ID] {
		return nexus.StaffIdentity{}, nexus.ErrStaffCredentialRejected
	}

	rows, err := m.db.Query(ctx,
		`SELECT p.membership_id::text,p.pin_hash,m.user_id::text,u.name,u.email,p.locked_until
		   FROM staff_pin_credentials p
		   JOIN memberships m ON m.id = p.membership_id
		   JOIN users u ON u.id = m.user_id
		  WHERE p.tenant_id = $1 AND p.active`, tenantID)
	if err != nil {
		return nexus.StaffIdentity{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var membershipID, hash, userID, name, email string
		var lockedUntil *time.Time
		if err := rows.Scan(&membershipID, &hash, &userID, &name, &email, &lockedUntil); err != nil {
			return nexus.StaffIdentity{}, err
		}
		if lockedUntil != nil && lockedUntil.After(time.Now()) {
			continue
		}
		if !auth.CheckPasswordHash(secret, hash) {
			continue
		}
		return nexus.StaffIdentity{
			UserID: userID, MembershipID: membershipID, Name: name, Email: email,
		}, nil
	}
	if err := rows.Err(); err != nil {
		return nexus.StaffIdentity{}, err
	}
	return nexus.StaffIdentity{}, nexus.ErrStaffCredentialRejected
}
