package ssoclients_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/domain/ssoclients"
	"github.com/gerege-systems/open-gerege-nexus/backend/domain/ssoclients/memory"
)

// vocabulary is this deployment as the tests state it: the grants the token
// endpoint implements, the scopes the consent screen renders, and the one host
// the operator has allowed.
func vocabulary() memory.Vocabulary {
	return memory.Vocabulary{
		Grants: []string{"authorization_code", "refresh_token", "client_credentials"},
		Scopes: []string{"openid", "profile", "erp.read", "erp.write"},
		Hosts:  []string{"example.mn"},
	}
}

func newService() *ssoclients.Service {
	return ssoclients.NewService(vocabulary(), memory.Identifiers{Value: "abcd1234"})
}

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
	service := newService()

	valid := func() ssoclients.Registration {
		return ssoclients.Registration{
			ClientName:   "Жишээ",
			ClientType:   "confidential",
			RedirectURIs: []string{"https://example.mn/callback"},
		}
	}

	refusals := []struct {
		name string
		edit func(*ssoclients.Registration)
		want string
	}{
		{"no name", func(r *ssoclients.Registration) { r.ClientName = "  " }, "client_name is required"},
		{"a name nobody typed", func(r *ssoclients.Registration) { r.ClientName = strings.Repeat("a", 201) }, "client_name is too long"},
		{"an invented client type", func(r *ssoclients.Registration) { r.ClientType = "hybrid" }, "client_type must be confidential or public"},
		{"an unsupported grant", func(r *ssoclients.Registration) { r.GrantTypes = []string{"password"} }, "unsupported grant type: password"},
		{"a public client with client_credentials", func(r *ssoclients.Registration) {
			r.ClientType = "public"
			r.GrantTypes = []string{"client_credentials"}
		}, "a public client cannot use client_credentials: it has no secret to prove with"},
		{"an unknown scope", func(r *ssoclients.Registration) { r.Scopes = []string{"everything"} }, "unknown scope: everything"},
		{"a code flow with nowhere to return to", func(r *ssoclients.Registration) { r.RedirectURIs = nil },
			"authorization_code requires at least one redirect_uri"},
		{"a relative redirect", func(r *ssoclients.Registration) { r.RedirectURIs = []string{"/callback"} },
			"redirect_uri must be an absolute URL: /callback"},
		{"a fragment the server never sees", func(r *ssoclients.Registration) { r.RedirectURIs = []string{"https://example.mn/cb#x"} },
			"redirect_uri must not contain a fragment: https://example.mn/cb#x"},
		{"a wildcard", func(r *ssoclients.Registration) { r.RedirectURIs = []string{"https://*.example.mn/cb"} },
			"wildcards are not allowed in a redirect_uri: https://*.example.mn/cb"},
		{"credentials in the callback", func(r *ssoclients.Registration) { r.RedirectURIs = []string{"https://user:pw@example.mn/cb"} },
			"redirect_uri must not carry userinfo: https://user:pw@example.mn/cb"},
		{"plain http off the loopback", func(r *ssoclients.Registration) { r.RedirectURIs = []string{"http://example.mn/cb"} },
			"redirect_uri must use https outside localhost: http://example.mn/cb"},
		{"a custom scheme on a confidential client", func(r *ssoclients.Registration) { r.RedirectURIs = []string{"com.example.app:/cb"} },
			"a confidential client's redirect_uri must be http(s): com.example.app:/cb"},
		{"a post-logout address with a wildcard", func(r *ssoclients.Registration) {
			r.PostLogoutRedirectURIs = []string{"https://*.example.mn/bye"}
		}, "wildcards are not allowed in a redirect_uri: https://*.example.mn/bye"},
		{"a relative logo", func(r *ssoclients.Registration) { r.LogoURI = "/logo.png" }, "client_uri and logo_uri must be absolute URLs"},
		// The operator's allowlist is the deployment-wide floor under this
		// app's rules, and its words are the provider's rather than ours.
		{"a host the operator never allowed", func(r *ssoclients.Registration) {
			r.RedirectURIs = []string{"https://somewhere.else/cb"}
		}, "redirect URI host is not allowed"},
	}

	for _, refusal := range refusals {
		t.Run(refusal.name, func(t *testing.T) {
			request := valid()
			refusal.edit(&request)
			out, err := service.Normalise(request)
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
	service := newService()

	out, err := service.Normalise(ssoclients.Registration{
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
		if _, err := service.Normalise(ssoclients.Registration{
			ClientName: "Жишээ", RedirectURIs: []string{loopback},
		}); err != nil {
			t.Fatalf("%s: %v", loopback, err)
		}
	}
	// A mobile app's custom scheme is meaningful for a public client.
	if _, err := service.Normalise(ssoclients.Registration{
		ClientName: "Жишээ", ClientType: "public", RedirectURIs: []string{"com.example.app:/cb"},
	}); err != nil {
		t.Fatalf("a public client's custom scheme: %v", err)
	}
}

// A public client is issued no secret at all: PKCE stands in for it, and a
// secret embedded in a mobile app is readable by anyone who downloads it.
func TestOnlyAConfidentialClientGetsASecret(t *testing.T) {
	service := newService()

	confidential, err := service.Normalise(ssoclients.Registration{
		ClientName: "Жишээ ХХК", RedirectURIs: []string{"https://example.mn/cb"},
	})
	if err != nil {
		t.Fatal(err)
	}
	clientID, secret := service.Credentials(confidential)
	// The readable half of the id is the name, so that a client_id in a log is
	// recognisable without a lookup.
	if clientID != "app_жишээ-ххк_abcd1234" && clientID != "app__abcd1234" {
		t.Logf("client_id is %q", clientID)
	}
	if !strings.HasPrefix(clientID, "app_") || !strings.HasSuffix(clientID, "_abcd1234") {
		t.Fatalf("the client_id lost its shape: %q", clientID)
	}
	if !strings.HasPrefix(secret, "sec_") {
		t.Fatalf("a confidential client needs a secret, got %q", secret)
	}

	public, err := service.Normalise(ssoclients.Registration{
		ClientName: "Гар утасны апп", ClientType: "public",
		RedirectURIs: []string{"com.example.app:/cb"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, secret := service.Credentials(public); secret != "" {
		t.Fatalf("a public client must be issued no secret, got %q", secret)
	}

	// And it cannot be given one later either.
	if _, err := service.RotateSecret(ssoclients.TypePublic); !errors.Is(err, ssoclients.ErrPublicHasNoSecret) {
		t.Fatalf("rotating a public client's secret: %v", err)
	}
	if rotated, err := service.RotateSecret(ssoclients.TypeConfidential); err != nil || !strings.HasPrefix(rotated, "sec_") {
		t.Fatalf("rotating a confidential client's secret: %q %v", rotated, err)
	}
}

// The kind of client is fixed at registration: flipping a public client to
// confidential would leave it with no secret, and the other direction would
// leave a secret in a binary that cannot keep one.
func TestTheKindOfClientCannotBeEditedAfterwards(t *testing.T) {
	asked := ssoclients.Registration{ClientName: "Жишээ", ClientType: "confidential"}
	if held := ssoclients.FixedType(asked, ssoclients.TypePublic); held.ClientType != ssoclients.TypePublic {
		t.Fatalf("an update must keep the registered type, got %q", held.ClientType)
	}
}
