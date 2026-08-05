package security

import (
	"net/http"
	"os"
	"strings"
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

// SanitizeSlug prevents path traversal attacks by requiring alphanumeric and hyphens only.
func IsValidSlug(slug string) bool {
	if slug == "" || len(slug) > 64 {
		return false
	}
	for _, ch := range slug {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return false
		}
	}
	return true
}

// SafeCORSOrigins returns whitelist of origins based on environment.
func SafeCORSOrigins() []string {
	envOrigins := os.Getenv("ALLOWED_ORIGINS")
	if envOrigins != "" {
		return strings.Split(envOrigins, ",")
	}
	return []string{"http://localhost:3000", "http://127.0.0.1:3000"}
}
