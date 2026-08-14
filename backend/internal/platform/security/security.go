package security

import (
	"net/http"
	"os"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/catalog"
)

// HeadersMiddleware injects HTTP security headers onto every response.
func HeadersMiddleware(next http.Handler) http.Handler {
	isProd := os.Getenv("ENVIRONMENT") == "production"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "1; mode=block")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		if isProd {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// IsValidSlug reports whether a slug is one the catalogue would accept.
//
// The rule moved to `backend/pkg/catalog` with the schema it belongs to: a slug
// is a store URL segment and a manifest filename, so what counts as one is part
// of the app-store contract rather than of this deployment's hardening. This
// forwards, because two packages each with their own idea of a valid slug would
// disagree exactly once — on the app that could be published and not installed.
func IsValidSlug(slug string) bool { return catalog.IsValidSlug(slug) }

// SafeCORSOrigins returns whitelist of origins based on environment.
//
// ALLOWED_ORIGINS is a comma-separated list, and people write it the way people
// write lists: "https://a.mn, https://b.mn". Splitting on the comma alone kept
// the leading space, and an origin with a space in it matches no Origin header
// ever sent, so every entry after the first was silently denied — a CORS
// failure in the browser with nothing wrong in the logs.
func SafeCORSOrigins() []string {
	origins := make([]string, 0)
	for _, origin := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) > 0 {
		return origins
	}
	return []string{"http://localhost:3000", "http://127.0.0.1:3000"}
}
