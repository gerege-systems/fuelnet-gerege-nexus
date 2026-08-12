package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/cache"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Which routes a stranger may reach.
//
// Every module is handed the root router — RegisterRoutes takes chi.Router, not
// a pre-gated group — so mounting a path outside the tenant gate is one line
// and looks exactly like mounting one inside it. That is deliberate: the
// registry module the App Store is becoming has to serve a public catalogue,
// and a platform that could not express "public" would have to run it as a
// second service, which is the arrangement being undone.
//
// The cost of allowing it is that a private route can become public by
// accident, and nothing about the diff would say so. This list is what says so:
// a route reachable without a session must be named here, and adding a name is
// a visible act in a review rather than a side effect of where a line was put.
//
// The entries are exact paths or prefixes ending in "/*". They are matched
// against chi's pattern, not against a request, so "/api/v1/registry/*" admits
// everything the registry module mounts under that prefix and nothing else.
var publicRoutes = []string{
	// Liveness and readiness. An orchestrator cannot hold a session.
	"/health",
	"/ready",
	"/metrics",

	// OIDC discovery and the keys that verify what this issuer signs. Public by
	// specification: a relying party reads them before anybody has signed in.
	"/.well-known/openid-configuration",
	"/.well-known/jwks.json",

	// The OAuth2 endpoints. Authorization and consent authenticate the end user
	// themselves; the token, introspection and revocation endpoints
	// authenticate the *client*, by secret or by PKCE, which is not a session
	// and does not come through authMiddleware.
	"/oauth2/auth",
	"/oauth2/token",
	"/oauth2/introspect",
	"/oauth2/revoke",
	"/oauth2/userinfo",

	// Signing in, and the identity flows that precede a session by definition.
	"/api/v1/auth/login",
	"/api/v1/auth/logout",
	"/api/v1/auth/eid/login",
	"/api/v1/auth/eid/start",
	"/api/v1/auth/eid/start-id",
	"/api/v1/auth/eid/poll",
	"/api/v1/auth/dan/login",

	// Two landings reached by people who are not signed in and may hold no
	// account here at all. In both cases a single-use reference in the query is
	// the whole authority — see handleIntegrationOAuthCallback and
	// handleVerifyLanded.
	"/api/v1/integrations/oauth/callback",
	"/api/v1/verify/landed",
}

// isPublic reports whether a chi pattern is on the list.
func isPublic(pattern string) bool {
	for _, allowed := range publicRoutes {
		if allowed == pattern {
			return true
		}
		if prefix, wildcard := strings.CutSuffix(allowed, "/*"); wildcard {
			if pattern == prefix || strings.HasPrefix(pattern, prefix+"/") {
				return true
			}
		}
	}
	return false
}

// walkRoutes lists every (method, pattern) the router serves.
func walkRoutes(t *testing.T, router chi.Routes) map[string][]string {
	t.Helper()
	found := map[string][]string{}
	err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		// chi reports a mounted subtree's pattern with a trailing /* on the
		// mount point; the leaf patterns come through separately.
		route = strings.TrimSuffix(route, "/*")
		if route == "" {
			route = "/"
		}
		found[route] = append(found[route], method)
		return nil
	})
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}
	return found
}

// A route not on the list must not be reachable without a session.
//
// This is asserted by making the request rather than by inspecting middleware:
// what matters is what a stranger gets, and a chain that looks right while
// answering 200 is exactly the failure worth catching. 401 or 403 both count —
// the first is "who are you", the second is "not you", and neither served
// anything.
//
// It runs against a real schema, so a handler that gets past the gate really
// runs and really answers — which is what makes a 200 here mean what it says.
func TestEveryRouteIsGatedUnlessItIsOnThePublicList(t *testing.T) {
	router := routerUnderTest(t)

	for pattern, methods := range walkRoutes(t, router) {
		if isPublic(pattern) {
			continue
		}
		for _, method := range methods {
			// A pattern with parameters needs a concrete path to be routed.
			target := concreteFor(pattern)
			req, err := http.NewRequest(method, target, strings.NewReader("{}"))
			if err != nil {
				t.Fatalf("%s %s: %v", method, target, err)
			}
			req.Header.Set("Content-Type", "application/json")

			rec := newRecorder()
			func() {
				// A handler reached without a database panics; the Recoverer
				// middleware turns that into a 500, which is still a refusal to
				// serve. Guarding here keeps a panic from ending the whole test.
				defer func() { _ = recover() }()
				router.(http.Handler).ServeHTTP(rec, req)
			}()

			if rec.Code == http.StatusOK || rec.Code == http.StatusCreated {
				t.Errorf("%s %s answered %d without a session; add it to publicRoutes if that is intended",
					method, pattern, rec.Code)
			}
		}
	}
}

// Nothing on the list may be there without still existing. A public route that
// has been renamed or removed leaves an entry that quietly widens the next
// route to take its name.
func TestThePublicListHasNoStaleEntries(t *testing.T) {
	served := walkRoutes(t, routerUnderTest(t))
	for _, allowed := range publicRoutes {
		if prefix, wildcard := strings.CutSuffix(allowed, "/*"); wildcard {
			matched := false
			for pattern := range served {
				if pattern == prefix || strings.HasPrefix(pattern, prefix+"/") {
					matched = true
					break
				}
			}
			if !matched {
				t.Errorf("publicRoutes carries %q but nothing is mounted under it", allowed)
			}
			continue
		}
		if _, ok := served[allowed]; !ok {
			t.Errorf("publicRoutes carries %q but the router does not serve it", allowed)
		}
	}
}

// routerUnderTest builds the real router, against the real schema.
//
// A stub router would assert only that this test's own wiring is gated. What is
// being checked is the routing table the process actually serves, so it is
// built by the same constructor the process uses.
//
//	AUTH_TEST_DATABASE_URL=postgres://... go test ./internal/platform/...
func routerUnderTest(t *testing.T) chi.Routes {
	t.Helper()
	dsn := os.Getenv("AUTH_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set AUTH_TEST_DATABASE_URL to a migrated test database to run the route policy tests")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	// The bundled catalogue, so the module routes under test are the ones this
	// build ships rather than whatever a registry is serving today.
	t.Setenv("APP_CATALOG_URL", "")
	// No Redis: the bus degrades to in-process, which is all a routing table
	// needs — nothing here crosses replicas.
	server, err := NewServer(pool, filepath.FromSlash("../../../catalog/apps.json"),
		cache.NewBus(context.Background(), nil))
	if err != nil {
		t.Fatalf("build the server: %v", err)
	}
	return server.router
}

// concreteFor turns a chi pattern into a path that will route: every {param}
// becomes a value that is syntactically plausible for the ones that are parsed.
var routeParam = regexp.MustCompile(`\{[^}]+\}`)

func concreteFor(pattern string) string {
	return routeParam.ReplaceAllStringFunc(pattern, func(param string) string {
		if strings.Contains(param, "id") || strings.Contains(param, "ID") {
			// A UUID, because a handler that parses one before checking the
			// session would otherwise answer 400 and hide what this asserts.
			return "00000000-0000-0000-0000-000000000000"
		}
		return "probe"
	})
}

func newRecorder() *httptest.ResponseRecorder { return httptest.NewRecorder() }
