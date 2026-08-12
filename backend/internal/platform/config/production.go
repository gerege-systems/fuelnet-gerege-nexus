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
	return validateSSOClient()
}

// validateSSOClient rejects a half-written relying-party configuration.
//
// The full check lives with the package that acts on it (ssoclient.Config's own
// Validate, which every deployment runs at startup). What is here is the part
// that is only a mistake *in production*: a plain-HTTP provider is allowed on
// the loopback interface so a developer can run two instances against each
// other, and that same allowance in production would be an identity handed
// across an unprotected hop.
func validateSSOClient() error {
	issuer := strings.TrimSpace(os.Getenv("SSO_CLIENT_ISSUER"))
	if issuer == "" {
		return nil
	}
	parsed, err := url.Parse(issuer)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("SSO_CLIENT_ISSUER must be an absolute HTTPS origin in production")
	}
	if strings.TrimSpace(os.Getenv("SSO_CLIENT_ID")) == "" {
		return fmt.Errorf("SSO_CLIENT_ID is required when SSO_CLIENT_ISSUER is set")
	}
	return nil
}

// RedirectHosts resolves the hostnames a redirect issued by this platform may
// send somebody to.
//
// The platform's own origin is always one of them, so the ordinary case needs
// no configuration. Anything else is named in envName, comma-separated, because
// sending a person somewhere else is a decision the operator makes once — not
// one a tenant administrator makes per request by typing a URL into a form.
//
// Outside production the loopback names are added, so a developer can point a
// verification link at the app they are running.
func RedirectHosts(envName string) []string {
	hosts := make([]string, 0, 4)
	if origin, err := url.Parse(strings.TrimSpace(os.Getenv("PUBLIC_ORIGIN"))); err == nil {
		if host := strings.ToLower(origin.Hostname()); host != "" {
			hosts = append(hosts, host)
		}
	}
	for _, extra := range strings.Split(os.Getenv(envName), ",") {
		if extra = strings.ToLower(strings.TrimSpace(extra)); extra != "" {
			hosts = append(hosts, extra)
		}
	}
	if !IsProduction() {
		hosts = append(hosts, "localhost", "127.0.0.1", "::1")
	}
	return hosts
}

// HostAllowed reports whether host is one of the allowed ones. The match is
// exact: a subdomain does not inherit its parent's trust, because on this
// deployment the neighbours under gerege.mn are other products.
func HostAllowed(host string, allowed []string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	for _, candidate := range allowed {
		if host == candidate {
			return true
		}
	}
	return false
}
