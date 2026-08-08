/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Persistence for the email verification service.
 */

package emailverify

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type store struct {
	db *pgxpool.Pool
}

const clientColumns = `
	id::text, tenant_id::text, name, key_prefix, status, hourly_limit,
	allowed_redirect_hosts, last_used_at, created_at, updated_at`

func scanClient(row pgx.Row) (*Client, error) {
	var c Client
	var hosts string
	if err := row.Scan(&c.ID, &c.TenantID, &c.Name, &c.KeyPrefix, &c.Status, &c.HourlyLimit,
		&hosts, &c.LastUsedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	c.AllowedRedirectHosts = splitHosts(hosts)
	return &c, nil
}

func splitHosts(raw string) []string {
	hosts := make([]string, 0)
	for _, entry := range strings.Split(raw, ",") {
		if entry = strings.TrimSpace(entry); entry != "" {
			hosts = append(hosts, entry)
		}
	}
	return hosts
}

// isUniqueViolation recognises the name constraint so the caller is told which
// rule they broke rather than being handed a driver message.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *store) insertClient(ctx context.Context, tenantID, name, keyPrefix, keyHash string, hourlyLimit int, hosts []string) (*Client, error) {
	row := s.db.QueryRow(ctx, `
		INSERT INTO email_verification_clients
			(tenant_id, name, key_prefix, key_hash, hourly_limit, allowed_redirect_hosts)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+clientColumns,
		tenantID, name, keyPrefix, keyHash, hourlyLimit, strings.Join(hosts, ","))
	client, err := scanClient(row)
	if isUniqueViolation(err) {
		return nil, ErrDuplicateName
	}
	return client, err
}

func (s *store) listClients(ctx context.Context, tenantID string) ([]Client, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+clientColumns+`
		FROM email_verification_clients
		WHERE tenant_id = $1
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clients := make([]Client, 0)
	for rows.Next() {
		client, scanErr := scanClient(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		clients = append(clients, *client)
	}
	return clients, rows.Err()
}

func (s *store) clientByKeyHash(ctx context.Context, keyHash string) (*Client, error) {
	row := s.db.QueryRow(ctx, `
		SELECT `+clientColumns+`
		FROM email_verification_clients
		WHERE key_hash = $1`, keyHash)
	client, err := scanClient(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUnauthorizedKey
	}
	return client, err
}

func (s *store) updateClient(ctx context.Context, tenantID, id, name string, status ClientStatus, hourlyLimit int, hosts []string) (*Client, error) {
	row := s.db.QueryRow(ctx, `
		UPDATE email_verification_clients
		SET name = $3, status = $4, hourly_limit = $5, allowed_redirect_hosts = $6, updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2
		RETURNING `+clientColumns,
		tenantID, id, name, status, hourlyLimit, strings.Join(hosts, ","))
	client, err := scanClient(row)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, ErrClientNotFound
	case isUniqueViolation(err):
		return nil, ErrDuplicateName
	}
	return client, err
}

func (s *store) deleteClient(ctx context.Context, tenantID, id string) error {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM email_verification_clients WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrClientNotFound
	}
	return nil
}

func (s *store) touchClient(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE email_verification_clients SET last_used_at = NOW() WHERE id = $1`, id)
	return err
}

const verificationColumns = `
	id::text, tenant_id::text, COALESCE(client_id::text, ''), source, purpose,
	email, redirect_url, status, expires_at, verified_at, created_at`

func scanVerification(row pgx.Row) (*Verification, error) {
	var v Verification
	if err := row.Scan(&v.ID, &v.TenantID, &v.ClientID, &v.Source, &v.Purpose, &v.Email,
		&v.RedirectURL, &v.Status, &v.ExpiresAt, &v.VerifiedAt, &v.CreatedAt); err != nil {
		return nil, err
	}
	return &v, nil
}

type newVerification struct {
	TenantID    string
	Client      *Client
	Source      string
	Purpose     string
	Email       string
	TokenHash   string
	RedirectURL string
	RequestedIP string
	ExpiresAt   time.Time
}

func (s *store) insertVerification(ctx context.Context, in newVerification) (*Verification, error) {
	var clientID *string
	if in.Client != nil {
		clientID = &in.Client.ID
	}
	row := s.db.QueryRow(ctx, `
		INSERT INTO email_verifications
			(tenant_id, client_id, source, purpose, email, token_hash, redirect_url,
			 requested_ip, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING `+verificationColumns,
		in.TenantID, clientID, in.Source, in.Purpose, in.Email, in.TokenHash,
		in.RedirectURL, in.RequestedIP, in.ExpiresAt)
	return scanVerification(row)
}

func (s *store) deleteVerification(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM email_verifications WHERE id = $1`, id)
	return err
}

// claimVerification marks a link used, and answers whether it was this call
// that used it.
//
// The condition lives in the UPDATE rather than in a preceding SELECT so that
// two clicks arriving at once cannot both succeed — a mail client that prefetches
// links would otherwise race the recipient and hand the win to whichever
// transaction read first.
func (s *store) claimVerification(ctx context.Context, tokenHash string) (*Verification, error) {
	row := s.db.QueryRow(ctx, `
		UPDATE email_verifications
		SET status = 'VERIFIED', verified_at = NOW()
		WHERE token_hash = $1 AND status = 'PENDING' AND expires_at > NOW()
		RETURNING `+verificationColumns, tokenHash)
	verification, err := scanVerification(row)
	if errors.Is(err, pgx.ErrNoRows) {
		// Used, expired, or never issued — one answer for all three, so a
		// caller holding a token learns only that it is no longer good.
		return nil, ErrLinkSpent
	}
	return verification, err
}

// countClientSends returns how many links a client issued inside the window and
// when the oldest of them was, which is what a Retry-After is computed from.
func (s *store) countClientSends(ctx context.Context, clientID string, since time.Time) (int, *time.Time, error) {
	var count int
	var oldest *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*), MIN(created_at)
		FROM email_verifications
		WHERE client_id = $1 AND created_at >= $2`, clientID, since).Scan(&count, &oldest)
	return count, oldest, err
}

func (s *store) countTenantSends(ctx context.Context, tenantID string, since time.Time) (int, *time.Time, error) {
	var count int
	var oldest *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*), MIN(created_at)
		FROM email_verifications
		WHERE tenant_id = $1 AND created_at >= $2`, tenantID, since).Scan(&count, &oldest)
	return count, oldest, err
}

func (s *store) lastSendTo(ctx context.Context, tenantID, email string) (*time.Time, error) {
	var last *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT MAX(created_at)
		FROM email_verifications
		WHERE tenant_id = $1 AND email = $2`, tenantID, email).Scan(&last)
	return last, err
}

func (s *store) recent(ctx context.Context, tenantID string, limit int) ([]Verification, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+verificationColumns+`
		FROM email_verifications
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Verification, 0, limit)
	for rows.Next() {
		verification, scanErr := scanVerification(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		list = append(list, *verification)
	}
	return list, rows.Err()
}

// stats counts a tenant's verifications in one pass. A pending row whose
// deadline has passed is reported as expired even before the sweep rewrites it,
// so the screen never claims somebody is still waiting on a dead link.
func (s *store) stats(ctx context.Context, tenantID string) (*Stats, error) {
	var st Stats
	err := s.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'VERIFIED'),
			COUNT(*) FILTER (WHERE status = 'PENDING' AND expires_at > NOW()),
			COUNT(*) FILTER (WHERE status = 'EXPIRED' OR (status = 'PENDING' AND expires_at <= NOW())),
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours')
		FROM email_verifications
		WHERE tenant_id = $1`, tenantID).
		Scan(&st.Total, &st.Verified, &st.Pending, &st.Expired, &st.Last24h)
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *store) expirePending(ctx context.Context) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE email_verifications
		SET status = 'EXPIRED'
		WHERE status = 'PENDING' AND expires_at <= NOW()`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *store) purgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM email_verifications WHERE created_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
