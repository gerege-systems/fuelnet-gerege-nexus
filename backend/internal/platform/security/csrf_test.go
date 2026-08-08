package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
)

func TestCSRFMiddleware(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://nexus.example")
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	for _, tc := range []struct {
		name, origin, fetchSite string
		cookie                  bool
		want                    int
	}{
		{"allowed cookie request", "https://nexus.example", "same-origin", true, http.StatusNoContent},
		{"foreign cookie request", "https://evil.example", "cross-site", true, http.StatusForbidden},
		{"bearer client", "https://evil.example", "cross-site", false, http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("Sec-Fetch-Site", tc.fetchSite)
			if tc.cookie {
				r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "token"})
			}
			w := httptest.NewRecorder()
			CSRFMiddleware(next).ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status=%d want=%d", w.Code, tc.want)
			}
		})
	}
}
