/*
 * Gerege Nexus — App Store registry
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

package appstore

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// Who is calling, as Gerege SSO describes them.
//
// The registry has no accounts of its own on purpose. Publishing is done by
// people who already have a Gerege identity, and giving this service a second
// password to look after would add a credential to steal and a recovery flow to
// get wrong for no gain.
type Caller struct {
	Subject    string
	Email      string
	Name       string
	TenantID   string
	TenantSlug string
	Admin      bool
}

// Verifier checks the identity tokens the developer console presents.
//
// It verifies rather than introspects: introspection would need this service to
// hold a client secret for the platform, and the console is a public client
// with no secret to share. An id_token is signed by the issuer, so its JWKS is
// enough — and the JWKS is public.
//
// Only RS256 is accepted and the algorithm is never taken from the token. That
// is the whole of the "alg confusion" class of JWT bugs: a verifier that lets
// the token choose the algorithm can be handed an HMAC signed with the public
// key, or "none".
type Verifier struct {
	issuer   string
	audience string
	client   *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time

	// adminSubjects and adminEmails are who may act on the review queue.
	// Deliberately configuration rather than a table: the first administrator
	// has to come from somewhere, and a bootstrap row that grants itself
	// administration is a worse answer than an environment variable an operator
	// sets deliberately.
	adminSubjects []string
	adminEmails   []string
}

// jwksTTL bounds how long a key set is trusted without asking again. Key
// rotation at the issuer should take effect within a coffee break, and the
// fetch is one request against a service on the same host.
const jwksTTL = 10 * time.Minute

func NewVerifier(issuer, audience string, adminSubjects, adminEmails []string) *Verifier {
	return &Verifier{
		issuer:        strings.TrimSuffix(issuer, "/"),
		audience:      audience,
		client:        &http.Client{Timeout: 10 * time.Second},
		keys:          map[string]*rsa.PublicKey{},
		adminSubjects: adminSubjects,
		adminEmails:   adminEmails,
	}
}

// idTokenClaims is the subset of an id_token this service reads.
type idTokenClaims struct {
	Issuer     string `json:"iss"`
	Subject    string `json:"sub"`
	Audience   any    `json:"aud"`
	Expiry     int64  `json:"exp"`
	IssuedAt   int64  `json:"iat"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	TenantID   string `json:"tenant_id"`
	TenantSlug string `json:"tenant_slug"`
}

// Verify checks a compact JWS and returns who it names.
func (v *Verifier) Verify(ctx context.Context, token string) (*Caller, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("not a compact JWS")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	if header.Alg != "RS256" {
		return nil, fmt.Errorf("unsupported token algorithm %q", header.Alg)
	}

	key, err := v.key(ctx, header.Kid)
	if err != nil {
		return nil, err
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return nil, errors.New("token signature does not verify")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}
	var claims idTokenClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}

	// A valid signature only says the issuer minted this. These say it was
	// minted for us, recently, and by the issuer we expect.
	if claims.Issuer != v.issuer {
		return nil, fmt.Errorf("token issued by %q, not %q", claims.Issuer, v.issuer)
	}
	if !audienceContains(claims.Audience, v.audience) {
		return nil, errors.New("token was issued for another application")
	}
	if claims.Expiry == 0 || time.Now().After(time.Unix(claims.Expiry, 0)) {
		return nil, errors.New("token has expired")
	}
	if claims.Subject == "" {
		return nil, errors.New("token names no subject")
	}

	return &Caller{
		Subject:    claims.Subject,
		Email:      claims.Email,
		Name:       claims.Name,
		TenantID:   claims.TenantID,
		TenantSlug: claims.TenantSlug,
		Admin: slices.Contains(v.adminSubjects, claims.Subject) ||
			(claims.Email != "" && slices.Contains(v.adminEmails, strings.ToLower(claims.Email))),
	}, nil
}

// audienceContains handles both shapes RFC 7519 allows for aud.
func audienceContains(aud any, expected string) bool {
	if expected == "" {
		return true
	}
	switch value := aud.(type) {
	case string:
		return value == expected
	case []any:
		for _, entry := range value {
			if text, ok := entry.(string); ok && text == expected {
				return true
			}
		}
	}
	return false
}

// key returns the issuer's signing key for a kid, refreshing the set when it is
// unknown or stale. An unknown kid is refetched once — that is what makes key
// rotation at the issuer take effect without restarting this service.
func (v *Verifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, known := v.keys[kid]
	fresh := time.Since(v.fetchedAt) < jwksTTL
	v.mu.RUnlock()
	if known && fresh {
		return key, nil
	}

	if err := v.refresh(ctx); err != nil {
		if known {
			// The issuer is unreachable but we have seen this key before.
			// Refusing here would take the console down for the length of an
			// outage that has nothing to do with it.
			return key, nil
		}
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	if key, known := v.keys[kid]; known {
		return key, nil
	}
	return nil, fmt.Errorf("no signing key %q at %s", kid, v.issuer)
}

func (v *Verifier) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.issuer+"/.well-known/jwks.json", nil)
	if err != nil {
		return err
	}
	res, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint answered %s", res.Status)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	var document struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(document.Keys))
	for _, jwk := range document.Keys {
		if jwk.Kty != "RSA" {
			continue
		}
		modulus, err := base64.RawURLEncoding.DecodeString(jwk.N)
		if err != nil {
			continue
		}
		exponent, err := base64.RawURLEncoding.DecodeString(jwk.E)
		if err != nil {
			continue
		}
		keys[jwk.Kid] = &rsa.PublicKey{
			N: new(big.Int).SetBytes(modulus),
			E: int(new(big.Int).SetBytes(exponent).Int64()),
		}
	}
	if len(keys) == 0 {
		return errors.New("jwks carries no usable RSA keys")
	}

	v.mu.Lock()
	v.keys, v.fetchedAt = keys, time.Now()
	v.mu.Unlock()
	return nil
}
