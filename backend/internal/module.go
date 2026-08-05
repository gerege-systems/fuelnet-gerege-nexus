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
}

// MenuDefinition defines a navigation menu item for an app module.
type MenuDefinition struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Path  string `json:"path"`
	Icon  string `json:"icon"`
	Order int    `json:"order"`
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
