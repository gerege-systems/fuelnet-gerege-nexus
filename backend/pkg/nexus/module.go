/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package nexus is the contract between this platform and the apps built on it.
 *
 * Everything in `internal/` is closed to other repositories — that is the Go
 * language rule, not a policy — and until this package existed the only way to
 * build a product on Gerege Nexus was to fork it. One fork per product means
 * every fix is applied once per product for ever, which is the failure this
 * package exists to prevent. What is here is what an app module needs in order
 * to be an app module, and nothing else:
 *
 *	Module            what a module is, and what the platform will ask of it
 *	Register          how a module joins the binary it is compiled into
 *	Dependency        what it needs installed first
 *	PermissionDefinition, MenuDefinition   what it declares to the platform
 *
 * The implementations stay in `internal/`. This package is a contract, and a
 * contract that also contained the machinery would drag the machinery into the
 * semver promise with it.
 *
 * # Stability
 *
 * This package is versioned with the module and does not break inside a major
 * version. That is the whole point of it: a distribution repository pins
 * `github.com/gerege-systems/open-gerege-nexus/backend vX.Y.Z` and expects its
 * modules to keep compiling until X changes. Anything that would break a
 * caller goes through deprecation and one major cycle — see CONTRIBUTING.
 *
 * The platform's own thirteen modules are written against this package rather
 * than against `internal/`. That is deliberate: an SDK its author does not use
 * is an SDK nobody has tried.
 */
package nexus

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Dependency describes an app module dependency and semver constraint.
type Dependency struct {
	ID                string `json:"id"`
	VersionConstraint string `json:"version_constraint"`
}

// PermissionDefinition defines an RBAC permission provided by a module.
type PermissionDefinition struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// AdminOnly withholds this permission from the default manager and user
	// roles when the app is installed. The tenant administrator still receives
	// it, and it can still be handed to any role by hand from Access control.
	//
	// It exists because the installer otherwise decides who gets a permission
	// by looking at the end of its code: anything ending `.read` is granted to
	// every member. That is a fine default for reading this organisation's own
	// rows and a bad one for reading somebody's national registry record, which
	// is a `.read` by grammar and an administrative act by consequence.
	AdminOnly bool `json:"admin_only,omitempty"`
}

// MenuDefinition defines a navigation menu item for an app module.
type MenuDefinition struct {
	ID       string `json:"id"`
	AppID    string `json:"app_id,omitempty"`
	AppName  string `json:"app_name,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
	Label    string `json:"label"`
	Path     string `json:"path,omitempty"`
	// ExternalURL is set instead of Path by an app that lives outside this
	// platform. The two are alternatives, not a pair: Path is a route in this
	// application and ExternalURL is somewhere else entirely, so the shell
	// renders one as a link it owns and the other as a link it is only
	// pointing at.
	ExternalURL string `json:"external_url,omitempty"`
	Icon        string `json:"icon"`
	Order       int    `json:"order"`

	// Labels holds per-locale overrides keyed by ISO 639-1 code. The menu API
	// resolves Label from the caller's locale before responding, so the client
	// never has to translate server-owned content.
	Labels map[string]string `json:"-"`
}

// LocalizedLabel returns the label for the requested locale, falling back to
// the default Label when no translation exists.
func (m MenuDefinition) LocalizedLabel(locale string) string {
	if label, ok := m.Labels[locale]; ok && label != "" {
		return label
	}
	return m.Label
}

// Module defines the contract every compile-time app module must implement.
//
// A module is a Go type with these seven methods and a constructor that calls
// Register. The platform owns HTTP, sessions, the database and the store; a
// module owns its own routes, its own tables and what it declares here.
//
// RegisterRoutes is handed the root router rather than a pre-scoped group, and
// the middleware it is given carries two decisions the platform has already
// made: whether this tenant has the app installed, and — for apps the platform
// knows a permission prefix for — whether the caller may make this kind of
// request. A module that wants a finer split applies its own permission
// middleware on top, which is what most of them do.
type Module interface {
	ID() string
	Name() string
	Version() string
	Dependencies() []Dependency
	Permissions() []PermissionDefinition
	Menus() []MenuDefinition
	RegisterRoutes(r chi.Router, tenantAuthMiddleware func(http.Handler) http.Handler)
}
