package internal

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
type Module interface {
	ID() string
	Name() string
	Version() string
	Dependencies() []Dependency
	Permissions() []PermissionDefinition
	Menus() []MenuDefinition
	RegisterRoutes(r chi.Router, tenantAuthMiddleware func(http.Handler) http.Handler)
}
