/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package developer_portal implements Developer Apps & OAuth2 SSO Client Portal Go module (io.example.developer_portal).
 */

package developer_portal

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appregistry"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/ssoprovider"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/tenant"
	"github.com/go-chi/chi/v5"
)

type DeveloperPortalModule struct {
	ssoProvider *ssoprovider.SSOProvider
}

// NewDeveloperPortalModule builds the module and registers it in the compile-time app registry.
func NewDeveloperPortalModule(ssoProvider *ssoprovider.SSOProvider) *DeveloperPortalModule {
	m := &DeveloperPortalModule{
		ssoProvider: ssoProvider,
	}
	appregistry.Register(m)
	return m
}

func (m *DeveloperPortalModule) ID() string      { return "io.example.developer_portal" }
func (m *DeveloperPortalModule) Name() string    { return "Developer Portal & OAuth2 SSO" }
func (m *DeveloperPortalModule) Version() string { return "1.0.0" }

func (m *DeveloperPortalModule) Dependencies() []internal.Dependency { return nil }

func (m *DeveloperPortalModule) Permissions() []internal.PermissionDefinition {
	return []internal.PermissionDefinition{
		{Code: "developer.read", Name: "Read Developer Apps", Description: "View registered OAuth2 client applications"},
		{Code: "developer.manage", Name: "Manage Developer Apps", Description: "Register and configure OAuth2 client applications"},
	}
}

func (m *DeveloperPortalModule) Menus() []internal.MenuDefinition {
	return []internal.MenuDefinition{
		{ID: "developer_apps", ParentID: "platform_tools", Label: "Developer Apps", Path: "/developer/apps", Icon: "code", Order: 10, Labels: map[string]string{"mn": "Хөгжүүлэгчийн аппууд", "ar": "تطبيقات المطورين", "zh": "开发者应用", "fr": "Applications développeur", "ru": "Приложения разработчика", "es": "Aplicaciones de desarrollador"}},
	}
}

func (m *DeveloperPortalModule) RegisterRoutes(r chi.Router, tenantAuthMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/developer/apps", func(dr chi.Router) {
		dr.Use(tenantAuthMiddleware)
		dr.Get("/", m.handleListApps)
		dr.Post("/", m.handleCreateApp)
	})
}

func (m *DeveloperPortalModule) handleListApps(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized tenant context"}`, http.StatusUnauthorized)
		return
	}

	clients, err := m.ssoProvider.ListTenantClients(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error":"failed to load OAuth clients"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(clients)
}

func (m *DeveloperPortalModule) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized tenant context"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		ClientName   string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
		Scopes       []string `json:"scopes"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
		return
	}

	if req.ClientName == "" {
		http.Error(w, `{"error":"client_name is required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Scopes) == 0 {
		req.Scopes = []string{"openid", "profile", "erp.read"}
	}
	allowedScopes := []string{"openid", "profile", "email", "erp.read", "erp.write"}
	for _, scope := range req.Scopes {
		if !slices.Contains(allowedScopes, scope) {
			http.Error(w, `{"error":"unsupported OAuth scope"}`, http.StatusBadRequest)
			return
		}
	}
	for _, raw := range req.RedirectURIs {
		u, parseErr := url.Parse(strings.TrimSpace(raw))
		localhostHTTP := u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")
		allowedTransport := u.Scheme == "https" || localhostHTTP
		if parseErr != nil || u.Host == "" || !allowedTransport {
			http.Error(w, `{"error":"redirect URI must use HTTPS (HTTP is allowed only for localhost)"}`, http.StatusBadRequest)
			return
		}
	}

	client := &ssoprovider.OAuth2Client{
		ClientName:   req.ClientName,
		RedirectURIs: req.RedirectURIs,
		Scopes:       req.Scopes,
		GrantTypes:   []string{"client_credentials"},
	}

	if err := m.ssoProvider.RegisterTenantClient(r.Context(), tenantID, client); err != nil {
		http.Error(w, `{"error":"failed to register OAuth client"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(client)
}
