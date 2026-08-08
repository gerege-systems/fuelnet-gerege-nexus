/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package emailverify proves that somebody controls an email address, for
 * every app module in the binary and for callers outside it.
 */

package emailverify

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/config"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/mailer"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Why this is platform furniture rather than an app module.
//
// Contacts wants an address proven before it trusts it, Documents wants it
// before a signing link leaves for an outsider, Gov Services wants it before it
// answers a citizen at one, and a platform running next to Gerege Nexus wants
// the same thing over HTTP for people who never sign in here at all. Every one
// of those is the same three steps — issue a single-use link, mail it, honour
// it once — and every copy of those three steps is another place to get token
// reuse or an open redirect wrong.
//
// So there is one implementation, in-process for modules (Service.Send) and
// over HTTP for everyone else (POST /api/v1/verify/send with a client key),
// writing one audit trail an administrator can actually read.

const (
	// KeyPrefix marks a verification client key on sight, so a key pasted into
	// the wrong field is recognisable — and so the send endpoint can tell a
	// client key from a session bearer token without trying both.
	KeyPrefix = "evk_"

	// keyPrefixLength is how much of the key is kept in the clear. It covers
	// "evk_" plus eight characters: enough to tell two keys apart on screen,
	// far short of enough to use one.
	keyPrefixLength = 12

	// DefaultTTL is how long a link stays good. Long enough for somebody who
	// reads their mail the next morning, short enough that a link forwarded
	// into a mailing-list archive stops working.
	DefaultTTL = 24 * time.Hour

	// MaxTTL bounds what a caller may ask for.
	MaxTTL = 7 * 24 * time.Hour

	// DefaultHourlyLimit is what a new client gets until an administrator says
	// otherwise.
	DefaultHourlyLimit = 60

	// TenantHourlyLimit caps sends that carry no client key — app modules and
	// the portal's own test send. Nothing here is metered per caller, so this
	// is what stands between a loop bug and a tenant's domain being classified
	// as a mail source worth blocking.
	TenantHourlyLimit = 500

	// ResendInterval is the pause enforced per recipient. Verification mail is
	// requested by whoever holds the key, addressed to somebody who did not ask
	// for it, which is the shape of a mail-bombing tool.
	ResendInterval = 60 * time.Second

	// Retention is how long the record of a verification is kept. It is an
	// audit trail of who was asked to prove what, not a mailing list.
	Retention = 90 * 24 * time.Hour
)

// ClientStatus mirrors the CHECK constraint on email_verification_clients.
type ClientStatus string

const (
	ClientActive   ClientStatus = "ACTIVE"
	ClientDisabled ClientStatus = "DISABLED"
)

// Status mirrors the CHECK constraint on email_verifications.
type Status string

const (
	StatusPending  Status = "PENDING"
	StatusVerified Status = "VERIFIED"
	StatusExpired  Status = "EXPIRED"
)

var (
	// ErrClientNotFound covers both "no such client" and "belongs to another
	// tenant". Every lookup is tenant-scoped, so a neighbour's id reads as
	// absent rather than as forbidden.
	ErrClientNotFound = errors.New("verification client not found")

	// ErrDuplicateName is the unique-name constraint, said for the person who
	// picked the name. The name is what a key is revoked by during an incident.
	ErrDuplicateName = errors.New("a verification client with this name already exists")

	// ErrUnauthorizedKey is returned for an unknown, malformed or disabled key.
	// The three are one answer on purpose: telling a caller their key exists
	// but is switched off confirms the key.
	ErrUnauthorizedKey = errors.New("unknown or disabled verification client key")

	// ErrLinkSpent covers a link that was already followed, has expired, or
	// never existed. A caller holding a token learns only that it is no longer
	// good for anything.
	ErrLinkSpent = errors.New("this verification link is no longer valid")

	// ErrMailUnavailable means the message could not even be accepted for
	// delivery, so no link was issued. It is not a silent drop.
	ErrMailUnavailable = errors.New("verification mail could not be queued for delivery")
)

// InvalidError is a caller mistake worth quoting back: a malformed address, a
// redirect that is not allowed. It maps to 400.
type InvalidError struct{ msg string }

func (e *InvalidError) Error() string { return e.msg }

func invalid(format string, args ...any) error {
	return &InvalidError{msg: fmt.Sprintf(format, args...)}
}

// RateLimitedError carries how long the caller should wait, so the handler can
// answer 429 with a Retry-After somebody can actually obey.
type RateLimitedError struct {
	RetryAfter time.Duration
	msg        string
}

func (e *RateLimitedError) Error() string { return e.msg }

// Client is one issued key. The key itself exists exactly once, in the response
// that creates it; Secret is empty on every later read because the database
// holds only its hash.
type Client struct {
	ID                   string       `json:"id"`
	TenantID             string       `json:"tenant_id"`
	Name                 string       `json:"name"`
	KeyPrefix            string       `json:"key_prefix"`
	Status               ClientStatus `json:"status"`
	HourlyLimit          int          `json:"hourly_limit"`
	AllowedRedirectHosts []string     `json:"allowed_redirect_hosts"`
	LastUsedAt           *time.Time   `json:"last_used_at,omitempty"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`

	// Secret is the full key, returned once at creation and never again.
	Secret string `json:"secret,omitempty"`
}

// Verification is one issued link. The token is not a field: the row holds a
// hash of it, so reading this table does not let anyone complete somebody
// else's verification.
type Verification struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	ClientID    string     `json:"client_id,omitempty"`
	Source      string     `json:"source"`
	Purpose     string     `json:"purpose,omitempty"`
	Email       string     `json:"email"`
	RedirectURL string     `json:"redirect_url,omitempty"`
	Status      Status     `json:"status"`
	ExpiresAt   time.Time  `json:"expires_at"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Request is what a caller asks for. Every field except Email is optional.
type Request struct {
	Email string

	// RedirectURL is where the recipient lands after following the link. Empty
	// means the platform answers the click itself.
	RedirectURL string

	// Purpose is the caller's own label — "signup", "contact_invite" — carried
	// through to the audit trail and back to the caller. It is not interpreted.
	Purpose string

	// Source names who asked: an app module id for an in-process call, the
	// client name for a keyed one, "portal" for the settings screen. It is kept
	// on the row so history survives the client being deleted.
	Source string

	// Locale picks the language of the mail. Empty falls back to Mongolian, the
	// platform's source language.
	Locale string

	// ClientIP is recorded for the audit trail. Empty is fine.
	ClientIP string

	// TTL overrides DefaultTTL, bounded by MaxTTL.
	TTL time.Duration
}

// Stats is the Overview screen's header.
type Stats struct {
	Total       int     `json:"total"`
	Verified    int     `json:"verified"`
	Pending     int     `json:"pending"`
	Expired     int     `json:"expired"`
	Last24h     int     `json:"last_24h"`
	VerifiedPct float64 `json:"verified_pct"`
}

// Overview is what one screen needs in one request.
type Overview struct {
	Stats  Stats          `json:"stats"`
	Recent []Verification `json:"recent"`
	// SendURL and ConfirmURL are what a developer has to paste into their own
	// integration, derived here from PUBLIC_ORIGIN so the screen never has to
	// guess the deployment's public address.
	SendURL    string `json:"send_url"`
	ConfirmURL string `json:"confirm_url"`
	// MailConfigured is false when SMTP was never set up, in which case mail is
	// logged instead of sent. Saying so beats letting an administrator conclude
	// the feature is broken.
	MailConfigured bool `json:"mail_configured"`
}

// Mailer is the outbound side. mailer.AsyncOTPMailer satisfies it: the send is
// queued rather than performed inline, because an SMTP conversation is slower
// than any caller should have to wait on a request that has already succeeded.
type Mailer interface {
	EnqueueMessage(msg mailer.EmailMessage) bool
}

// Service is the whole capability. One instance is built by the platform server
// and handed to whatever needs it — handlers, and app modules by constructor.
type Service struct {
	store  *store
	mailer Mailer
}

// NewService builds the service over a database pool and an outbound mailer.
func NewService(db *pgxpool.Pool, m Mailer) *Service {
	return &Service{store: &store{db: db}, mailer: m}
}

// PublicOrigin is the address a recipient's browser can reach.
//
// It is read from PUBLIC_ORIGIN rather than from the incoming request for the
// same reason the OAuth redirect URI is: the link goes in an email that outlives
// the request, and taking the host from a request lets a forged Host header
// point every verification link at somebody else's server.
func PublicOrigin() string {
	origin := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_ORIGIN")), "/")
	if origin == "" {
		origin = "http://localhost:8080"
	}
	return origin
}

// ConfirmURL is the endpoint the mailed link points at.
func ConfirmURL() string { return PublicOrigin() + "/api/v1/verify/confirm" }

// SendURL is the endpoint an outside caller posts to.
func SendURL() string { return PublicOrigin() + "/api/v1/verify/send" }

// Send issues a link and queues the mail. This is the entry point for app
// modules: a module takes *Service in its constructor, the way gov_services
// takes the integration manager, and calls this with its own app id as Source.
//
//	v, err := m.emailVerify.Send(ctx, tenantID, emailverify.Request{
//	    Email:   contact.Email,
//	    Source:  m.ID(),
//	    Purpose: "contact_invite",
//	})
func (s *Service) Send(ctx context.Context, tenantID string, req Request) (*Verification, error) {
	return s.send(ctx, tenantID, nil, req)
}

// SendForClient is the keyed path. The client's own hourly allowance and
// redirect allowlist apply, and the send is attributed to it.
func (s *Service) SendForClient(ctx context.Context, client *Client, req Request) (*Verification, error) {
	if client == nil {
		return nil, ErrUnauthorizedKey
	}
	if req.Source == "" {
		req.Source = client.Name
	}
	return s.send(ctx, client.TenantID, client, req)
}

func (s *Service) send(ctx context.Context, tenantID string, client *Client, req Request) (*Verification, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, invalid("a tenant is required")
	}

	address, err := NormalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}

	var allowedHosts []string
	if client != nil {
		allowedHosts = client.AllowedRedirectHosts
	}
	redirect, err := ValidateRedirect(req.RedirectURL, allowedHosts)
	if err != nil {
		return nil, err
	}

	ttl := req.TTL
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if ttl > MaxTTL {
		return nil, invalid("a verification link may not live longer than %d hours", int(MaxTTL.Hours()))
	}

	if err := s.checkQuota(ctx, tenantID, client, address); err != nil {
		return nil, err
	}

	token, err := randomToken()
	if err != nil {
		return nil, err
	}

	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "platform"
	}

	verification, err := s.store.insertVerification(ctx, newVerification{
		TenantID:    tenantID,
		Client:      client,
		Source:      truncate(source, 128),
		Purpose:     truncate(strings.TrimSpace(req.Purpose), 64),
		Email:       address,
		TokenHash:   hashSecret(token),
		RedirectURL: redirect,
		RequestedIP: truncate(req.ClientIP, 64),
		ExpiresAt:   time.Now().Add(ttl),
	})
	if err != nil {
		return nil, err
	}

	message := composeMessage(address, ConfirmURL()+"?token="+url.QueryEscape(token), verification.ExpiresAt, req.Locale)
	if !s.mailer.EnqueueMessage(message) {
		// The row is the link. If the mail was never accepted for delivery,
		// leaving the row behind would show a verification on the Overview
		// screen that nobody was ever asked to complete.
		if delErr := s.store.deleteVerification(ctx, verification.ID); delErr != nil {
			slog.Error("emailverify: could not withdraw an unsent verification",
				"id", verification.ID, "error", delErr)
		}
		return nil, ErrMailUnavailable
	}

	if client != nil {
		if err := s.store.touchClient(ctx, client.ID); err != nil {
			// Losing "last used at" is not worth failing a send that happened.
			slog.Warn("emailverify: could not record client usage", "client_id", client.ID, "error", err)
		}
	}

	return verification, nil
}

// checkQuota applies the two limits that stand between this endpoint and a
// mail-bombing tool: how much one caller may send in an hour, and how often one
// recipient may be written to at all.
func (s *Service) checkQuota(ctx context.Context, tenantID string, client *Client, address string) error {
	since := time.Now().Add(-time.Hour)

	if client != nil {
		limit := client.HourlyLimit
		if limit <= 0 {
			limit = DefaultHourlyLimit
		}
		used, oldest, err := s.store.countClientSends(ctx, client.ID, since)
		if err != nil {
			return err
		}
		if used >= limit {
			return &RateLimitedError{
				RetryAfter: retryAfter(oldest),
				msg:        fmt.Sprintf("this client may send %d verifications per hour", limit),
			}
		}
	} else {
		used, oldest, err := s.store.countTenantSends(ctx, tenantID, since)
		if err != nil {
			return err
		}
		if used >= TenantHourlyLimit {
			return &RateLimitedError{
				RetryAfter: retryAfter(oldest),
				msg:        fmt.Sprintf("this tenant may send %d verifications per hour", TenantHourlyLimit),
			}
		}
	}

	last, err := s.store.lastSendTo(ctx, tenantID, address)
	if err != nil {
		return err
	}
	if last != nil {
		if wait := ResendInterval - time.Since(*last); wait > 0 {
			// The timestamp came from the database's clock and the comparison
			// is against this process's. A database a second ahead would
			// otherwise produce a wait longer than the interval itself, which
			// is a number we would then be asking a caller to obey.
			if wait > ResendInterval {
				wait = ResendInterval
			}
			return &RateLimitedError{
				RetryAfter: wait,
				msg:        "a verification was just sent to this address",
			}
		}
	}
	return nil
}

// retryAfter turns the oldest send inside the window into how long until it
// leaves the window and frees an allowance.
func retryAfter(oldest *time.Time) time.Duration {
	if oldest == nil {
		return time.Minute
	}
	wait := time.Until(oldest.Add(time.Hour))
	if wait < time.Minute {
		return time.Minute
	}
	// The timestamp is the database's, the comparison is this process's; a
	// skew between them must not turn into a wait longer than the window.
	if wait > time.Hour {
		return time.Hour
	}
	return wait
}

// Confirm honours a token exactly once.
//
// The claim is a single conditional UPDATE rather than a read followed by a
// write: two clicks arriving together — a mail client prefetching the link
// while the recipient also clicks it — must not both succeed.
func (s *Service) Confirm(ctx context.Context, token string) (*Verification, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrLinkSpent
	}
	verification, err := s.store.claimVerification(ctx, hashSecret(token))
	if err != nil {
		return nil, err
	}
	return verification, nil
}

// Authenticate resolves a presented client key.
//
// The lookup is by the key's SHA-256, which is what the table stores: the key
// never exists in the database, so a dump of it is not a set of working
// credentials. A disabled client is refused here, which is what makes disabling
// take effect on the next request rather than whenever something expires.
func (s *Service) Authenticate(ctx context.Context, key string) (*Client, error) {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, KeyPrefix) || len(key) < keyPrefixLength+8 {
		return nil, ErrUnauthorizedKey
	}
	client, err := s.store.clientByKeyHash(ctx, hashSecret(key))
	if err != nil {
		return nil, err
	}
	if client.Status != ClientActive {
		return nil, ErrUnauthorizedKey
	}
	return client, nil
}

// ListClients returns one tenant's keys, newest first. No secret comes back.
func (s *Service) ListClients(ctx context.Context, tenantID string) ([]Client, error) {
	return s.store.listClients(ctx, tenantID)
}

// ClientInput is the settings form.
type ClientInput struct {
	Name                 string
	Status               ClientStatus
	HourlyLimit          int
	AllowedRedirectHosts []string
}

// CreateClient issues a key. The returned Client carries Secret; it is the only
// time it exists outside the caller's own storage, which is why the screen
// shows it once and says so.
func (s *Service) CreateClient(ctx context.Context, tenantID string, in ClientInput) (*Client, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, invalid("a client name is required")
	}
	if len(name) > 255 {
		return nil, invalid("the client name is too long")
	}
	hosts, err := normalizeHosts(in.AllowedRedirectHosts)
	if err != nil {
		return nil, err
	}
	limit := in.HourlyLimit
	if limit <= 0 {
		limit = DefaultHourlyLimit
	}
	if limit > 100000 {
		return nil, invalid("the hourly limit is unreasonably high")
	}

	secret, err := randomKey()
	if err != nil {
		return nil, err
	}

	client, err := s.store.insertClient(ctx, tenantID, name, secret[:keyPrefixLength], hashSecret(secret), limit, hosts)
	if err != nil {
		return nil, err
	}
	client.Secret = secret
	return client, nil
}

// UpdateClient changes the name, the switch, the allowance or the allowlist. It
// never re-issues the key: a key that has to change is a key that is deleted and
// replaced, so nothing that was revoked can quietly come back.
func (s *Service) UpdateClient(ctx context.Context, tenantID, id string, in ClientInput) (*Client, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, invalid("a client name is required")
	}
	status := in.Status
	if status != ClientActive && status != ClientDisabled {
		return nil, invalid("status must be ACTIVE or DISABLED")
	}
	hosts, err := normalizeHosts(in.AllowedRedirectHosts)
	if err != nil {
		return nil, err
	}
	limit := in.HourlyLimit
	if limit <= 0 {
		limit = DefaultHourlyLimit
	}
	if limit > 100000 {
		return nil, invalid("the hourly limit is unreasonably high")
	}
	return s.store.updateClient(ctx, tenantID, id, name, status, limit, hosts)
}

// DeleteClient revokes a key permanently. The verifications it sent stay, with
// their source label, because the audit trail is about what was asked of whom.
func (s *Service) DeleteClient(ctx context.Context, tenantID, id string) error {
	return s.store.deleteClient(ctx, tenantID, id)
}

// Overview is the settings screen in one request.
func (s *Service) Overview(ctx context.Context, tenantID string, limit int) (*Overview, error) {
	if limit <= 0 || limit > 200 {
		limit = 25
	}
	stats, err := s.store.stats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	recent, err := s.store.recent(ctx, tenantID, limit)
	if err != nil {
		return nil, err
	}
	if stats.Total > 0 {
		stats.VerifiedPct = float64(stats.Verified) / float64(stats.Total) * 100
	}
	return &Overview{
		Stats:          *stats,
		Recent:         recent,
		SendURL:        SendURL(),
		ConfirmURL:     ConfirmURL(),
		MailConfigured: strings.TrimSpace(os.Getenv("SMTP_HOST")) != "",
	}, nil
}

// StartHousekeeping ages out links nobody followed and drops history past the
// retention window. Both run until ctx is cancelled at shutdown.
func (s *Service) StartHousekeeping(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			s.sweep(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *Service) sweep(ctx context.Context) {
	sweepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if expired, err := s.store.expirePending(sweepCtx); err != nil {
		slog.Warn("emailverify: could not expire stale verifications", "error", err)
	} else if expired > 0 {
		slog.Info("emailverify: expired verification links", "count", expired)
	}
	if purged, err := s.store.purgeOlderThan(sweepCtx, time.Now().Add(-Retention)); err != nil {
		slog.Warn("emailverify: could not purge verification history", "error", err)
	} else if purged > 0 {
		slog.Info("emailverify: purged verification history", "count", purged)
	}
}

// NormalizeEmail accepts one plain address and returns it lowercased.
//
// A display name ("Ops <ops@example.com>") is refused rather than unwrapped:
// the address is going into a header this package writes, and accepting
// anything with structure in it is how a second recipient gets appended.
func NormalizeEmail(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalid("an email address is required")
	}
	if len(trimmed) > 320 {
		return "", invalid("the email address is too long")
	}
	if strings.ContainsAny(trimmed, "\r\n<>,;\"") {
		return "", invalid("the email address is not a plain address")
	}
	address, err := mail.ParseAddress(trimmed)
	if err != nil || address.Name != "" || address.Address != trimmed {
		return "", invalid("the email address is not valid")
	}
	if strings.Count(address.Address, "@") != 1 {
		return "", invalid("the email address is not valid")
	}
	return strings.ToLower(address.Address), nil
}

// ValidateRedirect decides where a recipient may be sent after they click.
//
// This endpoint hands out redirects from the platform's own domain, so an
// unchecked destination makes Gerege Nexus an open redirector — the thing a
// phishing link needs to look like it came from here. Hence: HTTPS only (HTTP
// is tolerated for localhost outside production, where a developer has no
// certificate), no embedded credentials, and, when the client declares an
// allowlist, a host that is on it.
func ValidateRedirect(raw string, allowedHosts []string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		// No destination is a legitimate choice: the platform answers the
		// click itself and the caller reads the outcome from its own records.
		return "", nil
	}
	if len(trimmed) > 2048 {
		return "", invalid("the redirect URL is too long")
	}
	if strings.ContainsAny(trimmed, "\r\n") {
		return "", invalid("the redirect URL is not valid")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return "", invalid("the redirect URL must be absolute")
	}
	if parsed.User != nil {
		return "", invalid("the redirect URL must not carry credentials")
	}
	host := strings.ToLower(parsed.Hostname())
	localhost := host == "localhost" || host == "127.0.0.1" || host == "::1"
	switch {
	case parsed.Scheme == "https":
	case parsed.Scheme == "http" && localhost && !config.IsProduction():
	default:
		return "", invalid("the redirect URL must use HTTPS (HTTP is allowed only for localhost in development)")
	}
	if len(allowedHosts) > 0 {
		permitted := false
		for _, candidate := range allowedHosts {
			if strings.EqualFold(candidate, host) {
				permitted = true
				break
			}
		}
		if !permitted {
			return "", invalid("%s is not an allowed redirect host for this client", host)
		}
	}
	return trimmed, nil
}

// normalizeHosts cleans the allowlist an administrator typed. A whole URL is
// accepted and reduced to its host, because that is what people paste.
func normalizeHosts(raw []string) ([]string, error) {
	hosts := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, entry := range raw {
		candidate := strings.ToLower(strings.TrimSpace(entry))
		if candidate == "" {
			continue
		}
		if strings.Contains(candidate, "/") {
			parsed, err := url.Parse(candidate)
			if err != nil || parsed.Hostname() == "" {
				return nil, invalid("%q is not a host name", entry)
			}
			candidate = parsed.Hostname()
		}
		if strings.ContainsAny(candidate, " ,\r\n@:") {
			return nil, invalid("%q is not a host name", entry)
		}
		if _, duplicate := seen[candidate]; duplicate {
			continue
		}
		seen[candidate] = struct{}{}
		hosts = append(hosts, candidate)
	}
	return hosts, nil
}

// randomKey mints a client key: the prefix that identifies it plus 32 bytes of
// randomness, URL-safe so it survives being pasted into a header or a config
// file by hand.
func randomKey() (string, error) {
	raw, err := randomBytes(32)
	if err != nil {
		return "", err
	}
	return KeyPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

// randomToken mints the token that goes in the mailed link. Same size as a key:
// it is a bearer credential for exactly one act.
func randomToken() (string, error) {
	raw, err := randomBytes(32)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func randomBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("emailverify: could not read random bytes: %w", err)
	}
	return buf, nil
}

// hashSecret is what the database stores for both keys and tokens. SHA-256
// rather than bcrypt on purpose: these are full-entropy random strings, not
// passwords, so there is nothing for a work factor to defend against — and the
// hash is looked up on every send, where a deliberate slowdown would be the
// rate limit nobody asked for.
func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
