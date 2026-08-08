package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// ValidateProduction rejects dangerous implicit development defaults before
// the server opens a listener.
func ValidateProduction() error {
	if !IsProduction() {
		return nil
	}
	for _, name := range []string{"DATABASE_URL", "PUBLIC_ORIGIN", "ALLOWED_ORIGINS", "SSO_DEFAULT_CLIENT_SECRET"} {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			return fmt.Errorf("%s is required in production", name)
		}
	}
	origin, err := url.Parse(os.Getenv("PUBLIC_ORIGIN"))
	if err != nil || origin.Scheme != "https" || origin.Host == "" || origin.User != nil {
		return fmt.Errorf("PUBLIC_ORIGIN must be an absolute HTTPS origin")
	}
	if origin.Path != "" && origin.Path != "/" {
		return fmt.Errorf("PUBLIC_ORIGIN must not contain a path")
	}
	return nil
}
