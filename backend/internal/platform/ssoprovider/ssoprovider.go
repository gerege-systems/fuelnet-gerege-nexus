/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package ssoprovider provides an ORY Hydra-grade OAuth2 and OpenID Connect (OIDC)
 * Single Sign-On (SSO) Identity Provider engine.
 */

package ssoprovider

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OAuth2Client struct {
	ID string `json:"id"`
	// ClientSecret is populated only in the response that creates the client;
	// every later read redacts it (see Redacted).
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret,omitempty"`
	ClientName   string    `json:"client_name"`
	RedirectURIs []string  `json:"redirect_uris"`
	GrantTypes   []string  `json:"grant_types"`
	Scopes       []string  `json:"scopes"`
	CreatedAt    time.Time `json:"created_at"`
	secretHash   string
}

// Redacted returns a copy safe to serialise to API consumers.
func (c *OAuth2Client) Redacted() *OAuth2Client {
	if c == nil {
		return nil
	}
	clone := *c
	clone.ClientSecret = ""
	return &clone
}

type TokenIntrospection struct {
	Active    bool   `json:"active"`
	Scope     string `json:"scope,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	Sub       string `json:"sub,omitempty"`
	Exp       int64  `json:"exp,omitempty"`
	TokenType string `json:"token_type,omitempty"`
}

type SSOProvider struct {
	mu      sync.RWMutex
	issuer  string
	clients map[string]*OAuth2Client
	tokens  map[string]*TokenIntrospection
	db      *pgxpool.Pool
}

func NewSSOProvider(pools ...*pgxpool.Pool) *SSOProvider {
	issuer := os.Getenv("SSO_ISSUER")
	if issuer == "" {
		// The issuer is baked into every token already granted and into the
		// relying parties' configuration, so it is migrated deliberately rather
		// than tracked automatically. It moved from openerp.gerege.mn with the
		// Gerege Nexus rebrand; tokens issued under the old issuer stop
		// validating, so clients pinning it must be updated alongside.
		// Deployments override it with SSO_ISSUER.
		issuer = "https://nexus.gerege.mn"
	}

	provider := &SSOProvider{
		issuer:  issuer,
		clients: make(map[string]*OAuth2Client),
		tokens:  make(map[string]*TokenIntrospection),
	}
	if len(pools) > 0 {
		provider.db = pools[0]
	}

	// Bootstrap the built-in developer-portal client. Its secret used to be a
	// constant compiled into the binary — a published credential for every
	// deployment. It now comes from the environment, and outside production a
	// random one is generated and logged for local use.
	secret := os.Getenv("SSO_DEFAULT_CLIENT_SECRET")
	switch {
	case secret != "":
	case config.IsProduction():
		slog.Warn("SSO_DEFAULT_CLIENT_SECRET is not set — the built-in developer portal client is disabled")
		return provider
	default:
		secret = "sec_" + generateRandomString(32)
		slog.Info("generated development SSO client secret",
			"client_id", "gerege-dev-portal", "client_secret", secret)
	}

	// The built-in client is intentionally memory-backed. Tenant-created
	// clients are durable; this operator-owned bootstrap identity comes from
	// deployment configuration on every replica.
	provider.registerMemoryClient(&OAuth2Client{
		ID:           "cli_01",
		ClientID:     "gerege-dev-portal",
		ClientSecret: secret,
		ClientName:   "Gerege Developer Portal App",
		// A redirect URI is matched exactly against what the client sends, so a
		// client still sending the pre-rebrand openerp.gerege.mn callback is
		// rejected until it is updated. Add entries here rather than editing in
		// place if both origins must be accepted during a migration window.
		RedirectURIs: []string{"http://localhost:3000/callback", "https://nexus.gerege.mn/callback"},
		GrantTypes:   []string{"authorization_code", "client_credentials", "refresh_token"},
		// erp.read/erp.write are legacy compatibility scope names, kept through
		// the Gerege Nexus rebrand. They are protocol identifiers already held
		// in issued tokens and third-party client registrations; renaming them
		// would invalidate live grants.
		Scopes:    []string{"openid", "profile", "email", "erp.read", "erp.write"},
		CreatedAt: time.Now(),
	})

	return provider
}

func (s *SSOProvider) RegisterClient(client *OAuth2Client) {
	s.registerMemoryClient(client)
}

func (s *SSOProvider) registerMemoryClient(client *OAuth2Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if client.ID == "" {
		client.ID = generateRandomString(12)
	}
	if client.ClientID == "" {
		client.ClientID = "app_" + generateRandomString(16)
	}
	if client.ClientSecret == "" {
		client.ClientSecret = "sec_" + generateRandomString(32)
	}
	client.CreatedAt = time.Now()
	s.clients[client.ClientID] = client
}

// RegisterTenantClient persists one tenant's client. Only a SHA-256 digest of
// the high-entropy generated secret is stored, so a database leak cannot mint
// tokens as that client.
func (s *SSOProvider) RegisterTenantClient(ctx context.Context, tenantID string, client *OAuth2Client) error {
	if s.db == nil {
		s.registerMemoryClient(client)
		return nil
	}
	if client.ID == "" {
		client.ID = uuid.NewString()
	}
	if client.ClientID == "" {
		client.ClientID = "app_" + generateRandomString(16)
	}
	if client.ClientSecret == "" {
		client.ClientSecret = "sec_" + generateRandomString(32)
	}
	client.CreatedAt = time.Now()
	_, err := s.db.Exec(ctx, `INSERT INTO oauth2_clients
		(id,tenant_id,client_id,client_secret_hash,client_name,redirect_uris,grant_types,scopes,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, client.ID, tenantID, client.ClientID,
		hashSecret(client.ClientSecret), client.ClientName, client.RedirectURIs, client.GrantTypes, client.Scopes, client.CreatedAt)
	return err
}

// ListClients returns every registered client with its secret redacted. The
// secret is shown once, in the response to the call that creates the client.
func (s *SSOProvider) ListClients() []*OAuth2Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*OAuth2Client, 0, len(s.clients))
	for _, c := range s.clients {
		list = append(list, c.Redacted())
	}
	slices.SortFunc(list, func(a, b *OAuth2Client) int {
		return strings.Compare(a.ClientID, b.ClientID)
	})
	return list
}

func (s *SSOProvider) ListTenantClients(ctx context.Context, tenantID string) ([]*OAuth2Client, error) {
	if s.db == nil {
		return s.ListClients(), nil
	}
	rows, err := s.db.Query(ctx, `SELECT id::text,client_id,client_name,redirect_uris,grant_types,scopes,created_at
		FROM oauth2_clients WHERE tenant_id=$1 ORDER BY client_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]*OAuth2Client, 0)
	for rows.Next() {
		c := &OAuth2Client{}
		if err := rows.Scan(&c.ID, &c.ClientID, &c.ClientName, &c.RedirectURIs, &c.GrantTypes, &c.Scopes, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (s *SSOProvider) GetClient(clientID string) (*OAuth2Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client, ok := s.clients[clientID]
	if !ok {
		return nil, errors.New("client not found")
	}
	return client, nil
}

func (s *SSOProvider) getClient(ctx context.Context, clientID string) (*OAuth2Client, error) {
	if c, err := s.GetClient(clientID); err == nil {
		return c, nil
	}
	if s.db == nil {
		return nil, errors.New("client not found")
	}
	c := &OAuth2Client{}
	err := s.db.QueryRow(ctx, `SELECT id::text,client_id,client_secret_hash,client_name,redirect_uris,grant_types,scopes,created_at
		FROM oauth2_clients WHERE client_id=$1`, clientID).Scan(&c.ID, &c.ClientID, &c.secretHash, &c.ClientName, &c.RedirectURIs, &c.GrantTypes, &c.Scopes, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("client not found")
	}
	return c, err
}

// IssueToken generates a new OAuth2 Access Token
func (s *SSOProvider) IssueToken(clientID, sub, scope string, duration time.Duration) (string, error) {
	return s.issueToken(context.Background(), clientID, sub, scope, duration)
}

func (s *SSOProvider) issueToken(ctx context.Context, clientID, sub, scope string, duration time.Duration) (string, error) {
	token := "hydra_at_" + generateRandomString(32)
	exp := time.Now().Add(duration).Unix()
	if s.db != nil {
		_, err := s.db.Exec(ctx, `INSERT INTO oauth2_access_tokens(token_hash,client_id,subject,scope,expires_at)
			VALUES($1,$2,$3,$4,to_timestamp($5))`, hashSecret(token), clientID, sub, scope, exp)
		return token, err
	}

	s.mu.Lock()
	s.tokens[token] = &TokenIntrospection{
		Active:    true,
		Scope:     scope,
		ClientID:  clientID,
		Sub:       sub,
		Exp:       exp,
		TokenType: "Bearer",
	}
	s.mu.Unlock()

	return token, nil
}

// IntrospectToken inspects token validity (ORY Hydra standard)
func (s *SSOProvider) IntrospectToken(token string) *TokenIntrospection {
	return s.introspectToken(context.Background(), token)
}

func (s *SSOProvider) introspectToken(ctx context.Context, token string) *TokenIntrospection {
	if s.db != nil {
		res := &TokenIntrospection{TokenType: "Bearer"}
		err := s.db.QueryRow(ctx, `SELECT client_id,subject,scope,extract(epoch from expires_at)::bigint
			FROM oauth2_access_tokens WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>NOW()`, hashSecret(token)).
			Scan(&res.ClientID, &res.Sub, &res.Scope, &res.Exp)
		res.Active = err == nil
		return res
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tokens[token]
	if !ok {
		return &TokenIntrospection{Active: false}
	}

	if time.Now().Unix() > t.Exp {
		return &TokenIntrospection{Active: false}
	}

	return t
}

// RevokeToken invalidates an issued token
func (s *SSOProvider) RevokeToken(token string) bool {
	return s.revokeToken(context.Background(), token)
}

func (s *SSOProvider) revokeToken(ctx context.Context, token string) bool {
	return s.revokeTokenForClient(ctx, token, "")
}

func (s *SSOProvider) revokeTokenForClient(ctx context.Context, token, clientID string) bool {
	if s.db != nil {
		tag, err := s.db.Exec(ctx, `UPDATE oauth2_access_tokens SET revoked_at=NOW()
			WHERE token_hash=$1 AND revoked_at IS NULL AND ($2='' OR client_id=$2)`, hashSecret(token), clientID)
		return err == nil && tag.RowsAffected() > 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tokens[token]; ok {
		delete(s.tokens, token)
		return true
	}
	return false
}

// HTTP Handlers for OpenID Connect & OAuth2 Provider

// HandleOIDCDiscovery advertises only what this provider actually implements.
// Announcing authorization_code/refresh_token and RS256 id_tokens while the
// server issues opaque client_credentials tokens breaks conformant clients at
// the first redirect.
func (s *SSOProvider) HandleOIDCDiscovery(w http.ResponseWriter, r *http.Request) {
	doc := map[string]any{
		"issuer":                                s.issuer,
		"token_endpoint":                        s.issuer + "/oauth2/token",
		"jwks_uri":                              s.issuer + "/.well-known/jwks.json",
		"introspection_endpoint":                s.issuer + "/oauth2/introspect",
		"revocation_endpoint":                   s.issuer + "/oauth2/revoke",
		"response_types_supported":              []string{},
		"grant_types_supported":                 []string{"client_credentials"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"subject_types_supported":               []string{"public"},
		"scopes_supported":                      []string{"openid", "profile", "email", "phone", "erp.read", "erp.write"},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

// HandleJWKS returns an empty key set: access tokens are opaque and validated
// through /oauth2/introspect. The previous handler published a placeholder RSA
// key ("n": "mock_rsa_n_val") that no client could ever verify against.
func (s *SSOProvider) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{}})
}

// AuthenticateClient verifies client credentials in constant time.
//
// The previous condition — `clientSecret != "" && client.ClientSecret != secret`
// — skipped verification entirely when the request omitted the secret, so any
// caller who knew a client_id could mint access tokens.
func (s *SSOProvider) AuthenticateClient(clientID, clientSecret string) (*OAuth2Client, error) {
	return s.authenticateClient(context.Background(), clientID, clientSecret)
}

func (s *SSOProvider) authenticateClient(ctx context.Context, clientID, clientSecret string) (*OAuth2Client, error) {
	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, ErrInvalidClient
	}
	want := client.ClientSecret
	got := clientSecret
	if client.secretHash != "" {
		want, got = client.secretHash, hashSecret(clientSecret)
	}
	if subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
		return nil, ErrInvalidClient
	}
	return client, nil
}

// ErrInvalidClient is returned for any client authentication failure; callers
// must not distinguish "unknown client" from "wrong secret".
var ErrInvalidClient = errors.New("invalid_client")

func (s *SSOProvider) HandleTokenEndpoint(w http.ResponseWriter, r *http.Request) {
	// RFC 6749 §2.3.1 allows credentials either in the body or via HTTP Basic.
	clientID, clientSecret, hasBasic := r.BasicAuth()
	if !hasBasic {
		clientID = r.FormValue("client_id")
		clientSecret = r.FormValue("client_secret")
	}

	client, err := s.authenticateClient(r.Context(), clientID, clientSecret)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="oauth2"`)
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType == "" {
		grantType = "client_credentials"
	}
	if len(client.GrantTypes) > 0 && !slices.Contains(client.GrantTypes, grantType) {
		writeOAuthError(w, http.StatusBadRequest, "unauthorized_client",
			"grant type "+grantType+" is not enabled for this client")
		return
	}
	// Only client_credentials is implemented today; advertising support for
	// authorization_code without an /oauth2/auth endpoint would be a lie.
	if grantType != "client_credentials" {
		writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type",
			"only client_credentials is currently implemented")
		return
	}

	grantedScopes := client.Scopes
	if requested := strings.Fields(r.FormValue("scope")); len(requested) > 0 {
		for _, scope := range requested {
			if !slices.Contains(client.Scopes, scope) {
				writeOAuthError(w, http.StatusBadRequest, "invalid_scope", "requested scope is not enabled for this client")
				return
			}
		}
		grantedScopes = requested
	}
	scope := strings.Join(grantedScopes, " ")
	accessToken, err := s.issueToken(r.Context(), client.ClientID, client.ClientID, scope, tokenTTL)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to issue token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(tokenTTL.Seconds()),
		"scope":        scope,
	})
}

// HandleIntrospectEndpoint requires client authentication: an unauthenticated
// introspection endpoint lets anyone probe token validity (RFC 7662 §2.1).
func (s *SSOProvider) HandleIntrospectEndpoint(w http.ResponseWriter, r *http.Request) {
	clientID, clientSecret, hasBasic := r.BasicAuth()
	if !hasBasic {
		clientID = r.FormValue("client_id")
		clientSecret = r.FormValue("client_secret")
	}
	if _, err := s.authenticateClient(r.Context(), clientID, clientSecret); err != nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}

	res := s.introspectToken(r.Context(), r.FormValue("token"))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *SSOProvider) HandleRevokeEndpoint(w http.ResponseWriter, r *http.Request) {
	clientID, clientSecret, hasBasic := r.BasicAuth()
	if !hasBasic {
		clientID = r.FormValue("client_id")
		clientSecret = r.FormValue("client_secret")
	}
	if _, err := s.authenticateClient(r.Context(), clientID, clientSecret); err != nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}

	s.revokeTokenForClient(r.Context(), r.FormValue("token"), clientID)
	w.WriteHeader(http.StatusOK)
}

func writeOAuthError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": description,
	})
}

const tokenTTL = 1 * time.Hour

func hashSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// generateRandomString returns n hex characters of crypto/rand output.
//
// The old implementation drew n bytes and then truncated the 2n-character hex
// string back to n, silently halving the entropy of every generated secret.
func generateRandomString(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, (n+1)/2)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand never fails on supported platforms; treat it as fatal
		// rather than returning a predictable value.
		panic("ssoprovider: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)[:n]
}
