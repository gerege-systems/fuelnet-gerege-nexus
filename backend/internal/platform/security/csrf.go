package security

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
)

// CSRFMiddleware rejects cross-site state-changing requests authenticated by
// the session cookie. Bearer clients do not rely on ambient browser authority
// and are therefore unaffected.
func CSRFMiddleware(next http.Handler) http.Handler {
	allowed := make(map[string]struct{})
	for _, raw := range SafeCORSOrigins() {
		if u, err := url.Parse(raw); err == nil && u.Scheme != "" && u.Host != "" {
			allowed[u.Scheme+"://"+u.Host] = struct{}{}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		if _, err := r.Cookie(auth.SessionCookieName); err != nil {
			next.ServeHTTP(w, r)
			return
		}
		if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
			http.Error(w, `{"error":"forbidden: cross-site request"}`, http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil {
				http.Error(w, `{"error":"forbidden: invalid origin"}`, http.StatusForbidden)
				return
			}
			if _, ok := allowed[u.Scheme+"://"+u.Host]; !ok {
				http.Error(w, `{"error":"forbidden: origin not allowed"}`, http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}
