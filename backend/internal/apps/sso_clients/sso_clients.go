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
	"net/url"
	"slices"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/ssoprovider"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
)

type SSOClientsModule struct {
	sso *ssoprovider.SSOProvider
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
func New(sso *ssoprovider.SSOProvider) *SSOClientsModule {
	m := &SSOClientsModule{sso: sso}
	nexus.Register(m)
	return m
}

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

func (m *SSOClientsModule) Menus() []nexus.MenuDefinition {
	return []nexus.MenuDefinition{
		{ID: "sso_clients_apps", Label: "SSO clients", Path: "/sso-clients", Icon: "code", Order: 10, Labels: map[string]string{"mn": "SSO клиентүүд", "ar": "عملاء SSO", "zh": "SSO 客户端", "fr": "Clients SSO", "ru": "SSO-клиенты", "es": "Clientes SSO"}},
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

// appRequest is the create/update payload.
type appRequest struct {
	ClientName   string   `json:"client_name"`
	ClientURI    string   `json:"client_uri"`
	LogoURI      string   `json:"logo_uri"`
	ClientType   string   `json:"client_type"`
	RedirectURIs []string `json:"redirect_uris"`
	// Where the platform may return somebody after this application signs them
	// out of it. Optional: an application that never ends a session here needs
	// none, and one that does gets its return address matched exactly.
	PostLogoutRedirectURIs []string `json:"post_logout_redirect_uris"`
	GrantTypes             []string `json:"grant_types"`
	Scopes                 []string `json:"scopes"`
	Disabled               bool     `json:"disabled"`
}

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

	normalised, verr := normalise(&req)
	if verr != nil {
		nexus.Error(w, http.StatusBadRequest, verr.Error())
		return
	}

	client := &ssoprovider.Client{
		TenantID:               tenantID,
		ClientID:               "app_" + strings.ToLower(slugify(req.ClientName)) + "_" + ssoprovider.NewIdentifier(8),
		ClientName:             normalised.ClientName,
		ClientURI:              normalised.ClientURI,
		LogoURI:                normalised.LogoURI,
		ClientType:             normalised.ClientType,
		RedirectURIs:           normalised.RedirectURIs,
		PostLogoutRedirectURIs: normalised.PostLogoutRedirectURIs,
		GrantTypes:             normalised.GrantTypes,
		Scopes:                 normalised.Scopes,
	}

	// A public client is issued no secret at all: PKCE stands in for it,
	// because a secret embedded in a mobile app or an SPA is readable by
	// anyone who downloads it.
	var secret, secretHash string
	if client.ClientType != "public" {
		secret = "sec_" + ssoprovider.NewIdentifier(48)
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
	// The client type is fixed at registration: flipping a public client to
	// confidential would leave it with no secret, and the other direction
	// would leave a secret in a binary that cannot keep one.
	req.ClientType = existing.ClientType

	normalised, verr := normalise(&req)
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
	if client.IsPublic() {
		nexus.Error(w, http.StatusBadRequest, "a public client has no secret to rotate")
		return
	}

	secret := "sec_" + ssoprovider.NewIdentifier(48)
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

// normalise validates and cleans a create/update payload.
func normalise(req *appRequest) (*appRequest, error) {
	out := &appRequest{
		ClientName: strings.TrimSpace(req.ClientName),
		ClientURI:  strings.TrimSpace(req.ClientURI),
		LogoURI:    strings.TrimSpace(req.LogoURI),
		ClientType: strings.TrimSpace(req.ClientType),
	}

	if out.ClientName == "" {
		return nil, errors.New("client_name is required")
	}
	if len(out.ClientName) > 200 {
		return nil, errors.New("client_name is too long")
	}

	if out.ClientType == "" {
		out.ClientType = "confidential"
	}
	if out.ClientType != "confidential" && out.ClientType != "public" {
		return nil, errors.New("client_type must be confidential or public")
	}

	if out.GrantTypes = dedupe(req.GrantTypes); len(out.GrantTypes) == 0 {
		out.GrantTypes = []string{"authorization_code", "refresh_token"}
	}
	for _, grant := range out.GrantTypes {
		if !slices.Contains(ssoprovider.SupportedGrantTypes, grant) {
			return nil, errors.New("unsupported grant type: " + grant)
		}
		if grant == "client_credentials" && out.ClientType == "public" {
			return nil, errors.New("a public client cannot use client_credentials: it has no secret to prove with")
		}
	}

	if out.Scopes = dedupe(req.Scopes); len(out.Scopes) == 0 {
		out.Scopes = []string{"openid", "profile", "erp.read"}
	}
	for _, scope := range out.Scopes {
		if !ssoprovider.IsSupportedScope(scope) {
			return nil, errors.New("unknown scope: " + scope)
		}
	}
	out.RedirectURIs = dedupe(req.RedirectURIs)
	if slices.Contains(out.GrantTypes, "authorization_code") && len(out.RedirectURIs) == 0 {
		return nil, errors.New("authorization_code requires at least one redirect_uri")
	}
	for _, raw := range out.RedirectURIs {
		if err := validateRedirectURI(raw, out.ClientType); err != nil {
			return nil, err
		}
	}
	// The same rules: a post-logout address is matched exactly by the logout
	// endpoint, so a wildcard or a fragment registered here would be a target
	// that can never match, and a plain-HTTP one off the loopback would be a
	// person handed back over an unprotected hop.
	out.PostLogoutRedirectURIs = dedupe(req.PostLogoutRedirectURIs)
	for _, raw := range out.PostLogoutRedirectURIs {
		if err := validateRedirectURI(raw, out.ClientType); err != nil {
			return nil, err
		}
	}
	for _, raw := range []string{out.ClientURI, out.LogoURI} {
		if raw == "" {
			continue
		}
		if parsed, err := url.Parse(raw); err != nil || !parsed.IsAbs() {
			return nil, errors.New("client_uri and logo_uri must be absolute URLs")
		}
	}

	return out, nil
}

// validateRedirectURI enforces what the authorization endpoint will later match
// against. Nothing was checked before, so a client could be registered with a
// redirect target the flow would refuse — or worse, accept.
func validateRedirectURI(raw, clientType string) error {
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() {
		return errors.New("redirect_uri must be an absolute URL: " + raw)
	}
	// A fragment is never sent to the server and cannot be matched, so a
	// registration carrying one is a mistake worth naming now.
	if parsed.Fragment != "" || strings.Contains(raw, "#") {
		return errors.New("redirect_uri must not contain a fragment: " + raw)
	}
	if strings.Contains(raw, "*") {
		return errors.New("wildcards are not allowed in a redirect_uri: " + raw)
	}
	// Credentials in the URI are never part of a callback anyone means to
	// register, and they would be handed to whoever reads the browser's history.
	if parsed.User != nil {
		return errors.New("redirect_uri must not carry userinfo: " + raw)
	}

	switch parsed.Scheme {
	case "https":
		// The operator's host allowlist, when they have set one.
		return ssoprovider.ValidateRedirectURI(raw)
	case "http":
		// Plain HTTP is only ever safe on the loopback interface, which is how
		// native apps and local development receive the redirect.
		host := parsed.Hostname()
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return nil
		}
		return errors.New("redirect_uri must use https outside localhost: " + raw)
	default:
		// Custom schemes (com.example.app:/callback) are how a mobile app
		// receives a redirect, and are meaningful only for public clients.
		if clientType == "public" {
			return nil
		}
		return errors.New("a confidential client's redirect_uri must be http(s): " + raw)
	}
}

func dedupe(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" && !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

// slugify turns an application name into the readable half of its client_id.
func slugify(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
				b.WriteRune('-')
			}
		}
		if b.Len() >= 24 {
			break
		}
	}
	return strings.Trim(b.String(), "-")
}
