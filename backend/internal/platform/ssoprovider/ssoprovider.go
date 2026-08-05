package ssoprovider

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type OAuth2Client struct {
	ID           string    `json:"id"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	ClientName   string    `json:"client_name"`
	RedirectURIs []string  `json:"redirect_uris"`
	GrantTypes   []string  `json:"grant_types"`
	Scopes       []string  `json:"scopes"`
	CreatedAt    time.Time `json:"created_at"`
}

type TokenIntrospection struct {
	Active    bool     `json:"active"`
	Scope     string   `json:"scope,omitempty"`
	ClientID  string   `json:"client_id,omitempty"`
	Sub       string   `json:"sub,omitempty"`
	Exp       int64    `json:"exp,omitempty"`
	TokenType string   `json:"token_type,omitempty"`
}

type SSOProvider struct {
	mu           sync.RWMutex
	issuer       string
	clients      map[string]*OAuth2Client
	tokens       map[string]*TokenIntrospection
}

func NewSSOProvider() *SSOProvider {
	issuer := os.Getenv("SSO_ISSUER")
	if issuer == "" {
		issuer = "https://openerp.gerege.mn"
	}

	provider := &SSOProvider{
		issuer:  issuer,
		clients: make(map[string]*OAuth2Client),
		tokens:  make(map[string]*TokenIntrospection),
	}

	// Register default internal client app
	provider.RegisterClient(&OAuth2Client{
		ID:           "cli_01",
		ClientID:     "gerege-dev-portal",
		ClientSecret: "secret_gerege_dev_2026",
		ClientName:   "Gerege Developer Portal App",
		RedirectURIs: []string{"http://localhost:3000/callback", "https://openerp.gerege.mn/callback"},
		GrantTypes:   []string{"authorization_code", "client_credentials", "refresh_token"},
		Scopes:       []string{"openid", "profile", "email", "erp.read", "erp.write"},
		CreatedAt:    time.Now(),
	})

	return provider
}

func (s *SSOProvider) RegisterClient(client *OAuth2Client) {
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

func (s *SSOProvider) ListClients() []*OAuth2Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*OAuth2Client, 0, len(s.clients))
	for _, c := range s.clients {
		list = append(list, c)
	}
	return list
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

// IssueToken generates a new OAuth2 Access Token
func (s *SSOProvider) IssueToken(clientID, sub, scope string, duration time.Duration) (string, error) {
	token := "hydra_at_" + generateRandomString(32)
	exp := time.Now().Add(duration).Unix()

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
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tokens[token]; ok {
		delete(s.tokens, token)
		return true
	}
	return false
}

// HTTP Handlers for OpenID Connect & OAuth2 Provider

func (s *SSOProvider) HandleOIDCDiscovery(w http.ResponseWriter, r *http.Request) {
	config := map[string]any{
		"issuer":                                s.issuer,
		"authorization_endpoint":                s.issuer + "/oauth2/auth",
		"token_endpoint":                        s.issuer + "/oauth2/token",
		"userinfo_endpoint":                     s.issuer + "/oauth2/userinfo",
		"jwks_uri":                              s.issuer + "/.well-known/jwks.json",
		"introspection_endpoint":                s.issuer + "/oauth2/introspect",
		"revocation_endpoint":                   s.issuer + "/oauth2/revoke",
		"response_types_supported":             []string{"code", "token", "id_token"},
		"grant_types_supported":                []string{"authorization_code", "client_credentials", "refresh_token"},
		"subject_types_supported":              []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "phone", "erp.read", "erp.write"},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(config)
}

func (s *SSOProvider) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	jwks := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": "hydra-gerege-key-1",
				"n":   "mock_rsa_n_val",
				"e":   "AQAB",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jwks)
}

func (s *SSOProvider) HandleTokenEndpoint(w http.ResponseWriter, r *http.Request) {
	grantType := r.FormValue("grant_type")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	client, err := s.GetClient(clientID)
	if err != nil || (clientSecret != "" && client.ClientSecret != clientSecret) {
		http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
		return
	}

	if grantType == "" {
		grantType = "client_credentials"
	}

	accessToken, err := s.IssueToken(clientID, "user_admin_01", strings.Join(client.Scopes, " "), 1*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"server_error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        strings.Join(client.Scopes, " "),
		"id_token":     "hydra_id_token_mock",
	})
}

func (s *SSOProvider) HandleIntrospectEndpoint(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	res := s.IntrospectToken(token)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *SSOProvider) HandleRevokeEndpoint(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	s.RevokeToken(token)
	w.WriteHeader(http.StatusOK)
}

func generateRandomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n]
}
