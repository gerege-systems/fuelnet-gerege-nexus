package sso_clients

import (
	"strings"
	"testing"
)

// What a client registration is refused for, written down before the rules are
// moved.
//
// These are the checks that decide where this platform will hand somebody back
// after signing them in, and there were none of them at all until this app was
// written: a client could be registered with a redirect target the
// authorization endpoint would later refuse — or worse, accept. There was also
// no test. The table is the net: every message here is one an integrator reads,
// and the move that follows must not reword a single one.
func TestWhatARegistrationIsRefusedFor(t *testing.T) {
	// The operator's host allowlist, which every https callback is held to on
	// top of this app's own rules. Without it the default is nexus.gerege.mn
	// and every case below would fail on the host rather than on what it is
	// about.
	t.Setenv("OAUTH_REDIRECT_HOSTS", "example.mn")

	valid := func() *appRequest {
		return &appRequest{
			ClientName:   "Жишээ",
			ClientType:   "confidential",
			RedirectURIs: []string{"https://example.mn/callback"},
		}
	}

	refusals := []struct {
		name string
		edit func(*appRequest)
		want string
	}{
		{"no name", func(r *appRequest) { r.ClientName = "  " }, "client_name is required"},
		{"a name nobody typed", func(r *appRequest) { r.ClientName = strings.Repeat("a", 201) }, "client_name is too long"},
		{"an invented client type", func(r *appRequest) { r.ClientType = "hybrid" }, "client_type must be confidential or public"},
		{"an unsupported grant", func(r *appRequest) { r.GrantTypes = []string{"password"} }, "unsupported grant type: password"},
		{"a public client with client_credentials", func(r *appRequest) {
			r.ClientType = "public"
			r.GrantTypes = []string{"client_credentials"}
		}, "a public client cannot use client_credentials: it has no secret to prove with"},
		{"an unknown scope", func(r *appRequest) { r.Scopes = []string{"everything"} }, "unknown scope: everything"},
		{"a code flow with nowhere to return to", func(r *appRequest) { r.RedirectURIs = nil },
			"authorization_code requires at least one redirect_uri"},
		{"a relative redirect", func(r *appRequest) { r.RedirectURIs = []string{"/callback"} },
			"redirect_uri must be an absolute URL: /callback"},
		{"a fragment the server never sees", func(r *appRequest) { r.RedirectURIs = []string{"https://example.mn/cb#x"} },
			"redirect_uri must not contain a fragment: https://example.mn/cb#x"},
		{"a wildcard", func(r *appRequest) { r.RedirectURIs = []string{"https://*.example.mn/cb"} },
			"wildcards are not allowed in a redirect_uri: https://*.example.mn/cb"},
		{"credentials in the callback", func(r *appRequest) { r.RedirectURIs = []string{"https://user:pw@example.mn/cb"} },
			"redirect_uri must not carry userinfo: https://user:pw@example.mn/cb"},
		{"plain http off the loopback", func(r *appRequest) { r.RedirectURIs = []string{"http://example.mn/cb"} },
			"redirect_uri must use https outside localhost: http://example.mn/cb"},
		{"a custom scheme on a confidential client", func(r *appRequest) { r.RedirectURIs = []string{"com.example.app:/cb"} },
			"a confidential client's redirect_uri must be http(s): com.example.app:/cb"},
		{"a post-logout address with a wildcard", func(r *appRequest) {
			r.PostLogoutRedirectURIs = []string{"https://*.example.mn/bye"}
		}, "wildcards are not allowed in a redirect_uri: https://*.example.mn/bye"},
		{"a relative logo", func(r *appRequest) { r.LogoURI = "/logo.png" }, "client_uri and logo_uri must be absolute URLs"},
		// The operator's allowlist is the deployment-wide floor under this
		// app's rules, and its words are the provider's rather than ours.
		{"a host the operator never allowed", func(r *appRequest) {
			r.RedirectURIs = []string{"https://somewhere.else/cb"}
		}, "redirect URI host is not allowed"},
	}

	for _, refusal := range refusals {
		t.Run(refusal.name, func(t *testing.T) {
			request := valid()
			refusal.edit(request)
			out, err := normalise(request)
			if err == nil {
				t.Fatalf("expected a refusal, got %+v", out)
			}
			if err.Error() != refusal.want {
				t.Fatalf("got %q, want %q", err, refusal.want)
			}
		})
	}
}

// And what it fills in for somebody who left the form half empty.
func TestWhatARegistrationDefaultsTo(t *testing.T) {
	t.Setenv("OAUTH_REDIRECT_HOSTS", "example.mn")

	out, err := normalise(&appRequest{
		ClientName:   "  Жишээ  ",
		RedirectURIs: []string{"https://example.mn/cb", "https://example.mn/cb", " "},
	})
	if err != nil {
		t.Fatalf("a minimal registration should be accepted: %v", err)
	}
	if out.ClientName != "Жишээ" {
		t.Fatalf("the name was not trimmed: %q", out.ClientName)
	}
	if out.ClientType != "confidential" {
		t.Fatalf("a client with no type is confidential, got %q", out.ClientType)
	}
	if strings.Join(out.GrantTypes, ",") != "authorization_code,refresh_token" {
		t.Fatalf("the default grants changed: %v", out.GrantTypes)
	}
	if strings.Join(out.Scopes, ",") != "openid,profile,erp.read" {
		t.Fatalf("the default scopes changed: %v", out.Scopes)
	}
	// Duplicates and blanks are dropped rather than stored: a registration is
	// matched against exactly, and two copies of one address is one address.
	if len(out.RedirectURIs) != 1 {
		t.Fatalf("the redirect list was not cleaned up: %v", out.RedirectURIs)
	}

	// Localhost over plain HTTP is how a native app and a laptop receive the
	// redirect, and stays allowed.
	for _, loopback := range []string{"http://localhost:3000/cb", "http://127.0.0.1:3000/cb", "http://[::1]:3000/cb"} {
		if _, err := normalise(&appRequest{ClientName: "Жишээ", RedirectURIs: []string{loopback}}); err != nil {
			t.Fatalf("%s: %v", loopback, err)
		}
	}
	// A mobile app's custom scheme is meaningful for a public client.
	if _, err := normalise(&appRequest{
		ClientName: "Жишээ", ClientType: "public", RedirectURIs: []string{"com.example.app:/cb"},
	}); err != nil {
		t.Fatalf("a public client's custom scheme: %v", err)
	}
}
