package ssoprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// RP-initiated logout against a real database, for the same reason the rest of
// the flow tests are: the client's registered return addresses live in a
// column, and matching against them is the whole security of the endpoint.
//
//	OAUTH_TEST_DATABASE_URL=postgres://... go test ./internal/platform/ssoprovider/...

// recordingEnder stands in for the platform session store.
type recordingEnder struct{ revoked []string }

func (r *recordingEnder) Revoke(_ context.Context, token string) error {
	r.revoked = append(r.revoked, token)
	return nil
}

// endSession drives the endpoint with a session cookie, as a browser would.
func (f *fixture) endSession(t *testing.T, params url.Values) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/oauth2/logout?"+params.Encode(), nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "test-session"})
	rec := httptest.NewRecorder()
	f.provider.HandleEndSession(rec, req)
	return rec.Code, rec.Header().Get("Location")
}

func TestEndSessionEndsTheSessionAndReturnsToARegisteredAddress(t *testing.T) {
	f := newFixture(t, func(c *Client) {
		c.PostLogoutRedirectURIs = []string{"https://client.test/"}
	})
	ender := &recordingEnder{}
	f.provider.AttachSessionEnder(ender)

	status, location := f.endSession(t, url.Values{
		"client_id":                {f.client.ClientID},
		"post_logout_redirect_uri": {"https://client.test/"},
		"state":                    {"state-xyz"},
	})

	if status != http.StatusFound {
		t.Fatalf("status = %d, want a redirect", status)
	}
	if want := "https://client.test/?state=state-xyz"; location != want {
		t.Errorf("Location = %q, want %q", location, want)
	}
	if len(ender.revoked) != 1 || ender.revoked[0] != "test-session" {
		t.Errorf("revoked = %v, want the session the cookie names", ender.revoked)
	}
}

// The one thing this endpoint must never do. A logout URL is one a client hands
// out freely, so anybody can put a post_logout_redirect_uri on it; following an
// unregistered one would make the provider forward browsers wherever a phishing
// page asked.
func TestEndSessionRefusesAnUnregisteredReturnAddress(t *testing.T) {
	f := newFixture(t, func(c *Client) {
		c.PostLogoutRedirectURIs = []string{"https://client.test/"}
	})
	ender := &recordingEnder{}
	f.provider.AttachSessionEnder(ender)

	status, location := f.endSession(t, url.Values{
		"client_id":                {f.client.ClientID},
		"post_logout_redirect_uri": {"https://phishing.example/"},
	})

	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want the redirect refused", status)
	}
	if location != "" {
		t.Errorf("Location = %q, want no redirect at all", location)
	}
	// Still signed out. The person asked to be, and the argument about where to
	// send them afterwards is not a reason to leave the session alive.
	if len(ender.revoked) != 1 {
		t.Errorf("revoked = %v, want the session ended anyway", ender.revoked)
	}
}

// Naming no client is naming no list of return addresses, so there is nothing
// an unregistered address could be matched against.
func TestEndSessionRefusesAReturnAddressWithNoClient(t *testing.T) {
	f := newFixture(t)
	f.provider.AttachSessionEnder(&recordingEnder{})

	status, _ := f.endSession(t, url.Values{
		"post_logout_redirect_uri": {"https://client.test/"},
	})
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want the redirect refused", status)
	}
}

// A logout with nowhere to go still logs out, and lands the person on the
// platform they signed out of.
func TestEndSessionWithoutAReturnAddressLandsOnTheIssuer(t *testing.T) {
	f := newFixture(t)
	ender := &recordingEnder{}
	f.provider.AttachSessionEnder(ender)

	status, location := f.endSession(t, url.Values{})
	if status != http.StatusFound {
		t.Fatalf("status = %d, want a redirect", status)
	}
	if location != testIssuer+"/" {
		t.Errorf("Location = %q, want the issuer", location)
	}
	if len(ender.revoked) != 1 {
		t.Errorf("revoked = %v, want the session ended", ender.revoked)
	}
}

// A conformant relying party sends an id_token_hint and may send no client_id
// at all, so the hint has to be enough to work out whose return addresses to
// match against.
func TestEndSessionResolvesTheClientFromAnIDTokenHint(t *testing.T) {
	f := newFixture(t, func(c *Client) {
		c.PostLogoutRedirectURIs = []string{"https://client.test/"}
	})
	f.provider.AttachSessionEnder(&recordingEnder{})

	ctx := context.Background()
	idToken, err := f.provider.mintIDToken(ctx, f.client, f.tenantID, f.userID,
		[]string{"openid"}, "", time.Now())
	if err != nil {
		t.Fatalf("mint id_token: %v", err)
	}

	status, location := f.endSession(t, url.Values{
		"id_token_hint":            {idToken},
		"post_logout_redirect_uri": {"https://client.test/"},
	})
	if status != http.StatusFound {
		t.Fatalf("status = %d, want a redirect", status)
	}
	if location != "https://client.test/" {
		t.Errorf("Location = %q", location)
	}
}

// A hint that does not verify is ignored rather than believed. Believing one
// would let anybody name any client by writing a token, which is the whole
// point of checking the signature.
func TestEndSessionIgnoresAnUnverifiableHint(t *testing.T) {
	f := newFixture(t, func(c *Client) {
		c.PostLogoutRedirectURIs = []string{"https://client.test/"}
	})
	f.provider.AttachSessionEnder(&recordingEnder{})

	ctx := context.Background()
	idToken, err := f.provider.mintIDToken(ctx, f.client, f.tenantID, f.userID,
		[]string{"openid"}, "", time.Now())
	if err != nil {
		t.Fatalf("mint id_token: %v", err)
	}
	// Flip the *first* character of the signature. The last one carries unused
	// trailing bits — two base64url characters can decode to the same bytes
	// there, so changing it is not reliably a change to the signature.
	dot := strings.LastIndex(idToken, ".")
	first := idToken[dot+1]
	replacement := byte('A')
	if first == 'A' {
		replacement = 'B'
	}
	tampered := idToken[:dot+1] + string(replacement) + idToken[dot+2:]

	status, _ := f.endSession(t, url.Values{
		"id_token_hint":            {tampered},
		"post_logout_redirect_uri": {"https://client.test/"},
	})
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want the unverified hint to buy the caller nothing", status)
	}
}
