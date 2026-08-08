package emailverify

import (
	"strings"
	"testing"
	"time"
)

// The address goes into a header this package writes. Anything with structure
// in it — a display name, a comma, a newline — is a way to append a recipient
// or a header of the caller's choosing, so it is refused rather than cleaned up.
func TestNormalizeEmailRefusesAnythingButAPlainAddress(t *testing.T) {
	valid := map[string]string{
		"user@example.com":       "user@example.com",
		"  User@Example.COM  ":   "user@example.com",
		"first.last+tag@mail.mn": "first.last+tag@mail.mn",
	}
	for input, want := range valid {
		got, err := NormalizeEmail(input)
		if err != nil {
			t.Errorf("NormalizeEmail(%q) returned %v, want it accepted", input, err)
			continue
		}
		if got != want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", input, got, want)
		}
	}

	rejected := []string{
		"",
		"   ",
		"not-an-address",
		"Ops <ops@example.com>",
		"a@example.com, b@example.com",
		"a@example.com\r\nBcc: victim@example.com",
		"a@example.com\nSubject: forged",
		"a@b@example.com",
		strings.Repeat("a", 315) + "@example.com",
	}
	for _, input := range rejected {
		if got, err := NormalizeEmail(input); err == nil {
			t.Errorf("NormalizeEmail(%q) accepted it as %q, want a refusal", input, got)
		}
	}
}

// The confirm endpoint hands out redirects from the platform's own domain. An
// unchecked destination is what turns Gerege Nexus into the open redirector a
// phishing link wants to borrow.
func TestValidateRedirectRefusesWhatWouldMakeUsAnOpenRedirector(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")

	accepted := []string{
		"",
		"https://theirapp.com/verified",
		"https://theirapp.com/verified?next=%2Fhome",
		"http://localhost:3000/verified",
		"http://127.0.0.1:3000/verified",
	}
	for _, input := range accepted {
		if _, err := ValidateRedirect(input, nil); err != nil {
			t.Errorf("ValidateRedirect(%q) returned %v, want it accepted", input, err)
		}
	}

	rejected := []string{
		"http://theirapp.com/verified",
		"/verified",
		"theirapp.com/verified",
		"javascript:alert(1)",
		"https://user:pass@theirapp.com/verified",
		"https://theirapp.com/\r\nSet-Cookie: a=b",
		"data:text/html,<script>alert(1)</script>",
	}
	for _, input := range rejected {
		if _, err := ValidateRedirect(input, nil); err == nil {
			t.Errorf("ValidateRedirect(%q) accepted it, want a refusal", input)
		}
	}
}

// Outside development even localhost has to be HTTPS: a production deployment
// resolving "localhost" is resolving something on its own host.
func TestValidateRedirectRefusesPlainHTTPInProduction(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	if _, err := ValidateRedirect("http://localhost:3000/verified", nil); err == nil {
		t.Fatal("plain HTTP to localhost was accepted in production")
	}
	if _, err := ValidateRedirect("https://theirapp.com/verified", nil); err != nil {
		t.Fatalf("HTTPS was refused in production: %v", err)
	}
}

// A client may declare where its own recipients are allowed to land. That is
// what keeps one tenant's key from redirecting people to somebody else's site.
func TestValidateRedirectHonoursTheClientAllowlist(t *testing.T) {
	allowed := []string{"theirapp.com", "portal.theirapp.com"}

	if _, err := ValidateRedirect("https://theirapp.com/verified", allowed); err != nil {
		t.Errorf("an allowlisted host was refused: %v", err)
	}
	if _, err := ValidateRedirect("https://PORTAL.THEIRAPP.COM/verified", allowed); err != nil {
		t.Errorf("host matching must be case-insensitive: %v", err)
	}
	if _, err := ValidateRedirect("https://elsewhere.example/verified", allowed); err == nil {
		t.Error("a host outside the allowlist was accepted")
	}
	// A subdomain is not the host that was allowed. Matching by suffix would
	// let evil-theirapp.com through.
	if _, err := ValidateRedirect("https://evil-theirapp.com/verified", allowed); err == nil {
		t.Error("a host that merely looks like an allowlisted one was accepted")
	}
}

// An administrator pastes what they have: sometimes a host, sometimes a whole
// URL. Both mean the same allowlist entry.
func TestNormalizeHosts(t *testing.T) {
	got, err := normalizeHosts([]string{" TheirApp.com ", "https://portal.theirapp.com/return", "", "theirapp.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"theirapp.com", "portal.theirapp.com"}
	if len(got) != len(want) {
		t.Fatalf("normalizeHosts returned %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normalizeHosts returned %v, want %v", got, want)
		}
	}
	if _, err := normalizeHosts([]string{"not a host"}); err == nil {
		t.Error("a host name with a space in it was accepted")
	}
}

// A key is recognisable on sight, unique, and never equal to its own stored
// hash — the last of which is the whole point of storing the hash.
func TestKeysAreUniqueAndStoredOnlyAsHashes(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 64; i++ {
		key, err := randomKey()
		if err != nil {
			t.Fatalf("randomKey failed: %v", err)
		}
		if !strings.HasPrefix(key, KeyPrefix) {
			t.Fatalf("key %q does not carry the %q prefix", key, KeyPrefix)
		}
		if len(key) <= keyPrefixLength+8 {
			t.Fatalf("key %q is too short to be refused by Authenticate's length guard", key)
		}
		if _, duplicate := seen[key]; duplicate {
			t.Fatalf("randomKey repeated itself: %q", key)
		}
		seen[key] = struct{}{}

		hash := hashSecret(key)
		if hash == key {
			t.Fatal("the stored hash is the key itself")
		}
		if len(hash) != 64 {
			t.Fatalf("hash is %d characters, the column holds 64", len(hash))
		}
		if hashSecret(key) != hash {
			t.Fatal("hashing the same key twice gave two answers")
		}
	}
}

func TestTokensAreUnique(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 64; i++ {
		token, err := randomToken()
		if err != nil {
			t.Fatalf("randomToken failed: %v", err)
		}
		if strings.HasPrefix(token, KeyPrefix) {
			t.Fatalf("token %q looks like a client key", token)
		}
		if _, duplicate := seen[token]; duplicate {
			t.Fatalf("randomToken repeated itself: %q", token)
		}
		seen[token] = struct{}{}
	}
}

// The mail is read outside the product, by somebody who may never have seen it.
// Every language the platform claims to speak has to have wording for it, and
// the link has to survive into the body.
func TestComposeMessageSpeaksEveryPlatformLanguage(t *testing.T) {
	const link = "https://nexus.gerege.mn/api/v1/verify/confirm?token=abc"
	deadline := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

	for _, locale := range []string{"mn", "ar", "zh", "en", "fr", "ru", "es"} {
		msg := composeMessage("user@example.com", link, deadline, locale)
		if msg.Subject == "" {
			t.Errorf("%s has no subject", locale)
		}
		if !strings.Contains(msg.Body, link) {
			t.Errorf("%s body does not carry the link", locale)
		}
		if !strings.Contains(msg.Body, "2026-08-09 12:00 UTC") {
			t.Errorf("%s body does not state the deadline: %s", locale, msg.Body)
		}
		if msg.To != "user@example.com" {
			t.Errorf("%s addressed %q", locale, msg.To)
		}
	}

	// An unknown language falls back to Mongolian, the source language, rather
	// than to an empty subject.
	fallback := composeMessage("user@example.com", link, deadline, "ko")
	if fallback.Subject != wordings["mn"].subject {
		t.Errorf("an unsupported locale produced %q, want the Mongolian subject", fallback.Subject)
	}
}

func TestResultPageSpeaksEveryPlatformLanguage(t *testing.T) {
	for _, locale := range []string{"mn", "ar", "zh", "en", "fr", "ru", "es", "ko"} {
		for _, confirmed := range []bool{true, false} {
			title, body := ResultPage(locale, confirmed)
			if title == "" || body == "" {
				t.Errorf("ResultPage(%q, %v) returned an empty page", locale, confirmed)
			}
		}
	}
	confirmedTitle, _ := ResultPage("en", true)
	spentTitle, _ := ResultPage("en", false)
	if confirmedTitle == spentTitle {
		t.Error("a confirmed link and a spent one produce the same page")
	}
}

// Retry-After has to be a number the caller can obey. Zero, or a negative
// duration from a window that has already passed, is an instruction to retry
// immediately — which is what the limit exists to prevent.
func TestRetryAfterIsAlwaysWorthWaiting(t *testing.T) {
	if got := retryAfter(nil); got < time.Minute {
		t.Errorf("retryAfter(nil) = %v, want at least a minute", got)
	}
	longPast := time.Now().Add(-3 * time.Hour)
	if got := retryAfter(&longPast); got < time.Minute {
		t.Errorf("retryAfter(long past) = %v, want at least a minute", got)
	}
	justNow := time.Now()
	got := retryAfter(&justNow)
	if got < 50*time.Minute || got > time.Hour {
		t.Errorf("retryAfter(now) = %v, want close to the full hour", got)
	}
}

// PUBLIC_ORIGIN is what the link in the mail is built from. It is deliberately
// not taken from the request: the link outlives the request, and a forged Host
// header would otherwise point every verification at somebody else's server.
func TestLinksAreBuiltFromPublicOrigin(t *testing.T) {
	t.Setenv("PUBLIC_ORIGIN", "https://nexus.gerege.mn/")
	if got := ConfirmURL(); got != "https://nexus.gerege.mn/api/v1/verify/confirm" {
		t.Errorf("ConfirmURL() = %q", got)
	}
	if got := SendURL(); got != "https://nexus.gerege.mn/api/v1/verify/send" {
		t.Errorf("SendURL() = %q", got)
	}

	t.Setenv("PUBLIC_ORIGIN", "")
	if got := ConfirmURL(); !strings.HasPrefix(got, "http://localhost:8080/") {
		t.Errorf("unset PUBLIC_ORIGIN gave %q, want the local development default", got)
	}
}
