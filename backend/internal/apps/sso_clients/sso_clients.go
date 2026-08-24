/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Package sso_clients is where a tenant registers the systems that sign people
 * in through this platform (io.gerege.nexus.sso_clients): the management
 * surface for the OAuth2 clients that platform.ssoprovider then authenticates.
 *
 * It was called `developer_portal`, which named the wrong thing twice. There is
 * a real developer portal in this ecosystem — developer.gerege.mn, backed by
 * the appstore-gerege-nexus distribution — where a third party submits an app
 * to the store; an
 * administrator who wanted that and found this instead had no way to tell from
 * the name that they were in the wrong product. And what this actually is has
 * nothing to do with developers: it is CRUD over OAuth2 clients, done by
 * whoever runs the organisation's integrations.
 */

package sso_clients

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	domain "github.com/gerege-systems/open-gerege-nexus/backend/domain/ssoclients"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/tenant/ssoprovider"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
)

type SSOClientsModule struct {
	sso *ssoprovider.SSOProvider
	svc *domain.Service
}

// ID is the catalogue identifier.
const ID = "io.gerege.nexus.sso_clients"

// LegacyID is what this app was called before the rename. appcatalog resolves
// it to ID, so a registry that has not republished and an installation written
// before the migration both keep working.
//
// DEPRECATED: remove in vNEXT.
const LegacyID = "io.gerege.nexus.developer_portal"

// New builds the module and registers it in the compile-time app registry.
//
// The provider stays whole: issuing codes, exchanging tokens and holding the
// clients are one authorization server this deployment shares, and none of it
// is this app's decision. What the app decides — what a registration has to
// look like before the provider is allowed to hold it — is the service.
func New(sso *ssoprovider.SSOProvider) *SSOClientsModule {
	m := &SSOClientsModule{sso: sso, svc: domain.NewService(vocabulary{}, identifiers{})}
	nexus.Register(m)
	return m
}

// vocabulary is domain/ssoclients.Vocabulary over the authorization server:
// what it implements, what it renders, and what its operator allows.
type vocabulary struct{}

func (vocabulary) SupportedGrantTypes() []string { return ssoprovider.SupportedGrantTypes }

func (vocabulary) IsSupportedScope(scope string) bool { return ssoprovider.IsSupportedScope(scope) }

func (vocabulary) AllowedRedirect(raw string) error { return ssoprovider.ValidateRedirectURI(raw) }

// identifiers is the platform's randomness, which is the only source this
// deployment has and the only one an audit has looked at.
type identifiers struct{}

func (identifiers) New(length int) string { return ssoprovider.NewIdentifier(length) }

func (m *SSOClientsModule) ID() string { return ID }

// MenuPermission and RoutePermissionPrefix are this module's half of
// nexus.AccessPolicy — what the platform used to hold in a switch keyed by
// app ID, stated here so it survives the module moving to another repository.
func (m *SSOClientsModule) MenuPermission() string        { return "sso_clients.read" }
func (m *SSOClientsModule) RoutePermissionPrefix() string { return "sso_clients" }
func (m *SSOClientsModule) Name() string                  { return "SSO Clients" }
func (m *SSOClientsModule) Version() string               { return "2.0.0" }

func (m *SSOClientsModule) Dependencies() []nexus.Dependency { return nil }

func (m *SSOClientsModule) Permissions() []nexus.PermissionDefinition {
	return []nexus.PermissionDefinition{
		{Code: "sso_clients.read", Name: "Read SSO Clients", Description: "View registered OAuth2 client applications"},
		{Code: "sso_clients.manage", Name: "Manage SSO Clients", Description: "Register, configure and revoke OAuth2 client applications"},
	}
}

// Menus is the one screen that exists and the five that are planned.
//
// The five were a blueprint inside the platform — internal/tenant/menu/
// blueprints.go, a table keyed by app id that no module outside this repository
// could add itself to. They are declared here now, in the group each belongs
// to, which is what deleting that table needed. Their paths follow the same
// convention the blueprint produced, /module/<slug>/<id>, so nothing the shell
// routes to has moved.
func (m *SSOClientsModule) Menus() []nexus.MenuDefinition {
	return []nexus.MenuDefinition{
		{ID: "sso_clients_apps", Label: "SSO clients", Path: "/sso-clients", Icon: "code-2", Order: 10,
			Labels: map[string]string{"mn": "SSO клиентүүд", "ar": "عملاء SSO", "zh": "SSO 客户端", "fr": "Clients SSO", "ru": "SSO-клиенты", "es": "Clientes SSO"}},

		// API, OAuth and Webhook are product vocabulary, not prose. They stay
		// Latin even in the scripts that would otherwise transliterate them.
		{ID: "sso-clients_api-keys", Label: "API keys", Path: "/module/sso-clients/api-keys",
			Icon: "key-round", Order: 20,
			Labels: map[string]string{"mn": "API түлхүүр", "ar": "مفاتيح API", "zh": "API 密钥", "fr": "Clés API", "ru": "Ключи API", "es": "Claves API"}},
		// Access audit sits under Modules rather than Settings: it is something
		// you read, not something you configure.
		{ID: "sso-clients_audit", Label: "Access audit", Path: "/module/sso-clients/audit",
			Icon: "scroll-text", Order: 30,
			Labels: map[string]string{"mn": "Хандалтын аудит", "ar": "تدقيق الوصول", "zh": "访问审计", "fr": "Audit des accès", "ru": "Аудит доступа", "es": "Auditoría de acceso"}},
		// No Webhooks entry: Settings → Integrations already registers webhook
		// listeners with a target URL and a signing secret, and a second screen
		// over the same records would only disagree with the first eventually.

		{ID: "sso-clients_scopes", Label: "OAuth scopes", Path: "/module/sso-clients/scopes",
			Icon: "shield-check", Order: 10, Group: nexus.MenuGroupSettings,
			Labels: map[string]string{"mn": "OAuth scope", "ar": "نطاقات OAuth", "zh": "OAuth 权限范围", "fr": "Portées OAuth", "ru": "Области OAuth", "es": "Ámbitos OAuth"}},
		{ID: "sso-clients_redirects", Label: "Redirect policies", Path: "/module/sso-clients/redirects",
			Icon: "route", Order: 20, Group: nexus.MenuGroupSettings,
			Labels: map[string]string{"mn": "Redirect бодлого", "ar": "سياسات إعادة التوجيه", "zh": "重定向策略", "fr": "Politiques de redirection", "ru": "Политики перенаправления", "es": "Políticas de redirección"}},
		{ID: "sso-clients_signing-keys", Label: "Signing keys", Path: "/module/sso-clients/signing-keys",
			Icon: "key-square", Order: 30, Group: nexus.MenuGroupSettings,
			Labels: map[string]string{"mn": "Гарын үсгийн түлхүүр", "ar": "مفاتيح التوقيع", "zh": "签名密钥", "fr": "Clés de signature", "ru": "Ключи подписи", "es": "Claves de firma"}},
	}
}

// RegisterRoutes mounts the API twice: once at the app's own name, and once at
// the name it used to have.
//
// A dual mount rather than a redirect, which is what the organisation rename
// used: nothing here moved between the platform and the app, so both trees are
// the same handlers behind the same gate, and mounting twice states that more
// plainly than a table of rewrites would. The gate middleware carries both the
// app installation check and the sso_clients.read / sso_clients.manage split,
// which platform.appRequestPermission derives from the HTTP method.
//
// The old tree stays for a release because an OAuth2 client is somebody else's
// integration: whoever registered it is not necessarily reachable, and a 404 on
// the screen where they would fix it is a poor way to tell them.
func (m *SSOClientsModule) RegisterRoutes(r chi.Router, tenantAuthMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/sso-clients", m.routes(tenantAuthMiddleware))
	// DEPRECATED: remove in vNEXT.
	r.Route("/api/v1/developer", m.routes(tenantAuthMiddleware))
}

func (m *SSOClientsModule) routes(tenantAuthMiddleware func(http.Handler) http.Handler) func(chi.Router) {
	return func(dr chi.Router) {
		dr.Use(tenantAuthMiddleware)

		dr.Get("/scopes", m.handleListScopes)
		dr.Get("/endpoints", m.handleEndpoints)

		dr.Get("/audit", m.handleAudit)
		dr.Get("/signing-keys", m.handleSigningKeys)

		dr.Route("/apps", func(ar chi.Router) {
			ar.Get("/", m.handleListApps)
			ar.Post("/", m.handleCreateApp)
			ar.Get("/{clientID}", m.handleGetApp)
			ar.Put("/{clientID}", m.handleUpdateApp)
			ar.Delete("/{clientID}", m.handleDeleteApp)
			ar.Post("/{clientID}/rotate-secret", m.handleRotateSecret)
			// Revocation is a mutation, so the gate middleware maps it to
			// sso_clients.manage the same way a delete is.
			ar.Delete("/{clientID}/tokens", m.handleRevokeTokens)
			ar.Delete("/{clientID}/consents/{userID}", m.handleWithdrawConsent)
		})
	}
}

// handleListApps returns the clients belonging to the caller's tenant.
//
// The previous implementation read the tenant out of the context, threw it
// away, and then called a provider method that returned every client on the
// platform — so any tenant could enumerate every other tenant's integrations.
func (m *SSOClientsModule) handleListApps(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	clients, err := m.sso.Store().ListClients(r.Context(), tenantID)
	if err != nil {
		slog.Error("failed to list oauth2 clients", "error", err, "tenant_id", tenantID)
		nexus.Error(w, http.StatusInternalServerError, "could not load applications")
		return
	}
	nexus.JSON(w, http.StatusOK, clients)
}

func (m *SSOClientsModule) handleGetApp(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	client, err := m.sso.Store().GetTenantClient(r.Context(), tenantID, chi.URLParam(r, "clientID"))
	if errors.Is(err, ssoprovider.ErrNotFound) {
		nexus.Error(w, http.StatusNotFound, "application not found")
		return
	}
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not load the application")
		return
	}
	nexus.JSON(w, http.StatusOK, client)
}

// appRequest is the create/update payload, which is the domain's registration:
// one struct rather than two, so the request shape and the thing the rules
// check cannot drift apart.
type appRequest = domain.Registration

func (m *SSOClientsModule) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}
	claims, _ := nexus.UserFromContext(r.Context())

	var req appRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		nexus.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}

	normalised, verr := m.svc.Normalise(req)
	if verr != nil {
		nexus.Error(w, http.StatusBadRequest, verr.Error())
		return
	}

	clientID, secret := m.svc.Credentials(normalised)
	client := &ssoprovider.Client{
		TenantID:               tenantID,
		ClientID:               clientID,
		ClientName:             normalised.ClientName,
		ClientURI:              normalised.ClientURI,
		LogoURI:                normalised.LogoURI,
		ClientType:             normalised.ClientType,
		RedirectURIs:           normalised.RedirectURIs,
		PostLogoutRedirectURIs: normalised.PostLogoutRedirectURIs,
		GrantTypes:             normalised.GrantTypes,
		Scopes:                 normalised.Scopes,
	}

	// Only the digest is stored, and only when there is a secret at all: a
	// public client is issued none.
	secretHash := ""
	if secret != "" {
		secretHash = ssoprovider.HashSecret(secret)
	}

	created, err := m.sso.Store().CreateClient(r.Context(), client, secretHash, claims.UserID)
	if err != nil {
		slog.Error("failed to create an oauth2 client", "error", err, "tenant_id", tenantID)
		nexus.Error(w, http.StatusInternalServerError, "could not register the application")
		return
	}

	// The only time the secret is ever readable. Every later read redacts it,
	// because the database holds a digest and cannot reproduce it.
	created.Secret = secret
	nexus.JSON(w, http.StatusCreated, created)
}

func (m *SSOClientsModule) handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	existing, err := m.sso.Store().GetTenantClient(r.Context(), tenantID, chi.URLParam(r, "clientID"))
	if errors.Is(err, ssoprovider.ErrNotFound) {
		nexus.Error(w, http.StatusNotFound, "application not found")
		return
	}
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not load the application")
		return
	}

	var req appRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		nexus.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}
	// The client type is fixed at registration.
	req = domain.FixedType(req, existing.ClientType)

	normalised, verr := m.svc.Normalise(req)
	if verr != nil {
		nexus.Error(w, http.StatusBadRequest, verr.Error())
		return
	}

	existing.ClientName = normalised.ClientName
	existing.ClientURI = normalised.ClientURI
	existing.LogoURI = normalised.LogoURI
	existing.RedirectURIs = normalised.RedirectURIs
	existing.PostLogoutRedirectURIs = normalised.PostLogoutRedirectURIs
	existing.GrantTypes = normalised.GrantTypes
	existing.Scopes = normalised.Scopes
	existing.Disabled = req.Disabled

	updated, err := m.sso.Store().UpdateClient(r.Context(), tenantID, existing)
	if err != nil {
		slog.Error("failed to update an oauth2 client", "error", err)
		nexus.Error(w, http.StatusInternalServerError, "could not update the application")
		return
	}
	nexus.JSON(w, http.StatusOK, updated)
}

func (m *SSOClientsModule) handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	err := m.sso.Store().DeleteClient(r.Context(), tenantID, chi.URLParam(r, "clientID"))
	if errors.Is(err, ssoprovider.ErrNotFound) {
		nexus.Error(w, http.StatusNotFound, "application not found")
		return
	}
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not delete the application")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleRotateSecret issues a fresh secret and invalidates the old one. There
// was no way to do this before: a leaked secret meant deleting the integration
// and re-registering it under a new client_id.
func (m *SSOClientsModule) handleRotateSecret(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	clientID := chi.URLParam(r, "clientID")
	client, err := m.sso.Store().GetTenantClient(r.Context(), tenantID, clientID)
	if errors.Is(err, ssoprovider.ErrNotFound) {
		nexus.Error(w, http.StatusNotFound, "application not found")
		return
	}
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "could not load the application")
		return
	}
	secret, verr := m.svc.RotateSecret(client.ClientType)
	if verr != nil {
		nexus.Error(w, http.StatusBadRequest, verr.Error())
		return
	}

	if err := m.sso.Store().RotateClientSecret(r.Context(), tenantID, clientID, ssoprovider.HashSecret(secret)); err != nil {
		slog.Error("failed to rotate a client secret", "error", err)
		nexus.Error(w, http.StatusInternalServerError, "could not rotate the secret")
		return
	}

	client.Secret = secret
	nexus.JSON(w, http.StatusOK, client)
}

// handleAudit reports what this tenant's clients are actually doing: live
// tokens, standing consents, and when each credential was last exchanged.
//
// A credential nobody has used in months is the one worth deleting, and a
// consent nobody remembers granting is the one worth withdrawing; neither was
// visible anywhere before.
func (m *SSOClientsModule) handleAudit(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	activity, err := m.sso.Store().ClientActivityByTenant(r.Context(), tenantID)
	if err != nil {
		slog.Error("failed to load oauth2 client activity", "error", err, "tenant_id", tenantID)
		nexus.Error(w, http.StatusInternalServerError, "could not load activity")
		return
	}
	consents, err := m.sso.Store().ConsentsByTenant(r.Context(), tenantID, 200)
	if err != nil {
		slog.Error("failed to load oauth2 consents", "error", err, "tenant_id", tenantID)
		nexus.Error(w, http.StatusInternalServerError, "could not load consents")
		return
	}

	nexus.JSON(w, http.StatusOK, map[string]any{"clients": activity, "consents": consents})
}

// handleRevokeTokens invalidates every live token a client holds without
// deleting the registration, so a suspected leak can be contained while the
// integration keeps its client_id.
func (m *SSOClientsModule) handleRevokeTokens(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	clientID := chi.URLParam(r, "clientID")
	if _, err := m.sso.Store().GetTenantClient(r.Context(), tenantID, clientID); err != nil {
		nexus.Error(w, http.StatusNotFound, "application not found")
		return
	}

	revoked, err := m.sso.Store().RevokeClientTokens(r.Context(), tenantID, clientID)
	if err != nil {
		slog.Error("failed to revoke client tokens", "error", err, "client_id", clientID)
		nexus.Error(w, http.StatusInternalServerError, "could not revoke tokens")
		return
	}
	nexus.JSON(w, http.StatusOK, map[string]any{"revoked": revoked})
}

// handleWithdrawConsent removes one user's standing grant to a client.
func (m *SSOClientsModule) handleWithdrawConsent(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	err := m.sso.Store().WithdrawConsent(r.Context(), tenantID,
		chi.URLParam(r, "clientID"), chi.URLParam(r, "userID"))
	if errors.Is(err, ssoprovider.ErrNotFound) {
		nexus.Error(w, http.StatusNotFound, "consent not found")
		return
	}
	if err != nil {
		slog.Error("failed to withdraw consent", "error", err)
		nexus.Error(w, http.StatusInternalServerError, "could not withdraw the consent")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSigningKeys lists the keys the JWKS publishes, so an integrator can see
// which kid their library should be pinning and when it appeared. Only public
// metadata: the query never selects the private half.
func (m *SSOClientsModule) handleSigningKeys(w http.ResponseWriter, r *http.Request) {
	if _, ok := nexus.RequireTenant(w, r); !ok {
		return
	}

	keys, err := m.sso.Store().SigningKeys(r.Context())
	if err != nil {
		slog.Error("failed to load signing keys", "error", err)
		nexus.Error(w, http.StatusInternalServerError, "could not load signing keys")
		return
	}
	nexus.JSON(w, http.StatusOK, map[string]any{
		"keys":     keys,
		"jwks_uri": m.sso.Issuer() + "/.well-known/jwks.json",
	})
}

// handleListScopes gives the portal's scope picker the same vocabulary the
// consent screen renders, so the two cannot drift.
func (m *SSOClientsModule) handleListScopes(w http.ResponseWriter, r *http.Request) {
	nexus.JSON(w, http.StatusOK, map[string]any{
		"scopes":      ssoprovider.SupportedScopes,
		"grant_types": ssoprovider.SupportedGrantTypes,
	})
}

// handleEndpoints hands the portal the exact URLs an integrator has to paste
// into their client library, rather than making them assemble the origin.
func (m *SSOClientsModule) handleEndpoints(w http.ResponseWriter, r *http.Request) {
	issuer := m.sso.Issuer()
	nexus.JSON(w, http.StatusOK, map[string]string{
		"issuer":                 issuer,
		"discovery":              issuer + "/.well-known/openid-configuration",
		"jwks_uri":               issuer + "/.well-known/jwks.json",
		"authorization_endpoint": issuer + "/oauth2/auth",
		"token_endpoint":         issuer + "/oauth2/token",
		"userinfo_endpoint":      issuer + "/oauth2/userinfo",
		"introspection_endpoint": issuer + "/oauth2/introspect",
		"revocation_endpoint":    issuer + "/oauth2/revoke",
	})
}
