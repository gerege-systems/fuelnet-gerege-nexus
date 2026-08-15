/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package organisation is the organisation and the people in it.
 *
 * It was called `core` until the ecosystem split made the name a liability:
 * "core" is what the platform underneath every app is called, and an app named
 * after the floor it stands on cannot be told apart from it in a catalogue,
 * a permission code or an import path. What this module actually holds is the
 * organisation, its departments and its people — so that is what it is called.
 *
 * What is left here is the half that really is an app. The other half — the
 * tenant's legal profile and the signed-in person's own preferences — moved to
 * the platform (internal/platform/tenant_profile_handlers.go), because the
 * control plane, the XYP rail and an SSO consent screen all read the
 * organisation's registered name without caring which apps the tenant has, and
 * a screen those depend on cannot be one an administrator is able to remove.
 *
 * The shape follows Odoo's, because the distinctions it draws are real:
 *
 *	res.company   → tenants + tenant_profiles   what the organisation is  (platform)
 *	res.users     → users                       who the person is, anywhere (platform)
 *	hr.employee   → memberships                 who they are *here*         (here)
 *	hr.department → departments                 how it is arranged          (here)
 *
 * The middle line is the one worth keeping straight. A language preference
 * belongs to a person and follows them between organisations; a job title does
 * not. The same person can be a director in one tenant and a clerk in another,
 * so the title lives on the membership and the language lives on the user.
 *
 * Unlike Odoo's `base`, this one can be removed. Nothing imports it and no
 * other module's foreign keys point at a department, so a deployment that has
 * no use for an internal directory — a queue kiosk, a single-purpose portal —
 * should not be made to carry one. It is installed by default and uninstalling
 * it closes the gate without dropping a row.
 *
 * What this module does not do is authorisation. Roles and permissions already
 * exist and are already good (rbac + Settings → Access control); this names the
 * people those roles are handed to.
 */

package organisation

import (
	"net/http"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/contacts"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
)

// ID is the catalogue identifier. It is referenced by the platform, which
// installs this app for every tenant and refuses to disable it.
const ID = "io.gerege.nexus.organisation"

// LegacyID is what this app was called before the rename. It is not a second
// identity: appcatalog resolves it to ID when a catalogue or an installation
// still carries it, so a deployment that has not yet run the migration and a
// registry that has not yet republished both keep working.
//
// DEPRECATED: remove in vNEXT.
const LegacyID = "io.gerege.nexus.core"

type Module struct {
	db    nexus.DB
	perms nexus.PermissionStore

	// Who this organisation deals with, as opposed to who it is made of. It
	// was a second app in the store — see contacts.New for why half a
	// directory is not a product. Nil is allowed and means this deployment
	// builds the module without the register: the units and the people work,
	// and the contact routes are simply not mounted.
	contacts *contacts.Module
}

// contactRegister is the contacts half, or nil on a deployment built without it.
func New(p nexus.Platform, contactRegister *contacts.Module) *Module {
	m := &Module{db: p.DB(), perms: p.Permissions(), contacts: contactRegister}
	nexus.Register(m)
	registerReports()
	return m
}

func (m *Module) ID() string { return ID }

// Name is the question the app answers, which is "who".
//
// It was "Organisation & People" and sat in the store beside "Contacts", which
// was the same subject cut in half: the people inside and the people outside.
// A directory is what you get when neither half is optional.
func (m *Module) Name() string { return "Directory" }

// 2.0.0 because the app's shape changed. It absorbed another app, its
// permissions are what that app's used to be, and every tenant has it — this
// one is installed by default — so the contact register arrived everywhere at
// once.
func (m *Module) Version() string { return "2.0.0" }

func (m *Module) Dependencies() []nexus.Dependency { return nil }

// Permissions are deliberately two, not six.
//
// Reading who works here is something any member of the organisation may do —
// a directory nobody can open is not a directory. Changing the organisation's
// legal identity, its structure, or who belongs to it is administrative, and it
// is one decision rather than three: anybody who can add a person to a
// department can already put them anywhere in it.
func (m *Module) Permissions() []nexus.PermissionDefinition {
	return []nexus.PermissionDefinition{
		{Code: "organisation.read", Name: "Read Directory", Description: "View the organisation profile, its departments, its people and its contacts"},
		{Code: "organisation.manage", Name: "Manage Directory", Description: "Edit the organisation profile, its departments, its people and its contacts"},
	}
}

// Menus are the three screens this app is. No parent is named: the platform
// hangs an app's own menus under its Modules group, and a ParentID set here
// would be overwritten — which reads as a decision that was silently ignored.
func (m *Module) Menus() []nexus.MenuDefinition {
	return []nexus.MenuDefinition{
		{
			ID: "organisation_people", Label: "People",
			Path: "/organisation/people", Icon: "users", Order: 7,
			Labels: map[string]string{
				"mn": "Ажилтнууд", "ar": "الأشخاص", "zh": "人员",
				"fr": "Personnes", "ru": "Сотрудники", "es": "Personas",
			},
		},
		{
			ID: "organisation_departments", Label: "Departments",
			Path: "/organisation/departments", Icon: "network", Order: 6,
			Labels: map[string]string{
				"mn": "Хэлтэс, нэгж", "ar": "الأقسام", "zh": "部门",
				"fr": "Départements", "ru": "Подразделения", "es": "Departamentos",
			},
		},
		// The outside half. Its path is what it always was: an address people
		// have bookmarked is not ours to invalidate for a reorganisation they
		// did not ask for, and unlike the blueprint entries, a menu declared
		// here carries its own path rather than one built from a slug.
		{
			ID: "contacts", Label: "Contacts",
			Path: "/contacts", Icon: "users", Order: 10,
			Labels: map[string]string{
				"mn": "Харилцагчид", "ar": "جهات الاتصال", "zh": "联系人",
				"fr": "Contacts", "ru": "Контакты", "es": "Contactos",
			},
		},
	}
}

// RegisterRoutes mounts the app at its own name.
//
// The prefix is part of the rename: an app called `organisation` answering on
// /api/v1/core would keep the old name in the one place clients copy it from.
// The old prefix is still served for a release, but by the platform rather than
// from here — see tenantLegacyRoutes in internal/platform, which redirects it.
// It has to be the platform's, because half of what used to live under
// /api/v1/core is now a platform route that outlives this app being removed.
func (m *Module) RegisterRoutes(r chi.Router, tenantAuthMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/organisation", func(cr chi.Router) {
		cr.Use(tenantAuthMiddleware)
		read := nexus.RequirePermission(m.perms, "organisation.read")
		manage := nexus.RequirePermission(m.perms, "organisation.manage")

		// How it is arranged.
		cr.With(read).Get("/departments", m.handleListDepartments)
		cr.With(manage).Post("/departments", m.handleCreateDepartment)
		cr.With(manage).Put("/departments/{id}", m.handleUpdateDepartment)
		// Archiving and deleting are different acts and now say so. DELETE
		// used to archive, which left the screen with a Delete that did not
		// delete and no way to actually remove a unit created by mistake.
		cr.With(manage).Post("/departments/{id}/archive", m.handleArchiveDepartment)
		cr.With(manage).Post("/departments/{id}/restore", m.handleRestoreDepartment)
		cr.With(manage).Delete("/departments/{id}", m.handleDeleteDepartment)

		// Who is in it.
		cr.With(read).Get("/people", m.handleListPeople)
		cr.With(manage).Put("/people/{id}", m.handleUpdatePerson)
		cr.With(manage).Post("/people/{id}/deactivate", m.handleDeactivatePerson)
		cr.With(manage).Post("/people/{id}/reactivate", m.handleReactivatePerson)
	})

	// The contact register, mounted by the app that now owns it.
	//
	// It keeps its own prefix — /api/v1/contacts — deliberately. A path is a
	// contract with every client already written against it, and moving one to
	// make an internal reorganisation visible from outside costs other people's
	// afternoons and buys nothing. What merged is the product, not the wiring.
	if m.contacts != nil {
		m.contacts.RegisterRoutes(r, tenantAuthMiddleware)
	}
}
