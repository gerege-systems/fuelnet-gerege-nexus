/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * The reports app's own two tables in PostgreSQL: report_schedules and
 * report_grants. Every statement here was lifted out of a handler unchanged.
 *
 * What this layer decides is what an error means — no rows is a schedule that
 * is not this organisation's, a violated partial unique index is an agreement
 * that already exists — so that nothing above it has to know pgx.
 */
package postgres

import (
	"context"
	"errors"

	"github.com/gerege-systems/open-gerege-nexus/backend/domain/reports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DB is the handle this store needs, in pgx's own shape: the platform hands
// modules a pool that satisfies it as it is.
type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Tenancy is how this deployment scopes a query to an organisation.
//
// It is injected rather than imported because binding a tenant to a context is
// the platform's mechanism — row-level policies decided from the connection,
// see internal/platform/dbguard — and naming the SDK here would be the one
// import that puts it back inside the domain.
//
// Unbound is the platform path, outside the policies. Exactly one query needs
// it, and it is the one that deliberately looks across organisations: finding a
// tenant by its registration number.
type Tenancy struct {
	Bind    func(ctx context.Context, tenantID string) context.Context
	Unbound func(ctx context.Context) context.Context
}

type Store struct {
	db       DB
	tenantOf Tenancy
}

func New(db DB, tenancy Tenancy) *Store { return &Store{db: db, tenantOf: tenancy} }

func (s *Store) CreateSchedule(ctx context.Context, tenantID string, schedule reports.Schedule) (string, error) {
	var id string
	err := s.db.QueryRow(s.tenantOf.Bind(ctx, tenantID), `
		INSERT INTO report_schedules
		    (tenant_id, report_key, name, params, cron, format, recipients, active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, '')::uuid)
		RETURNING id`,
		tenantID, schedule.ReportKey, schedule.Name, schedule.Params, schedule.Cron,
		schedule.Format, schedule.Recipients, schedule.Active, schedule.CreatedBy).Scan(&id)
	return id, err
}

func (s *Store) UpdateSchedule(ctx context.Context, tenantID, id string, schedule reports.Schedule) (bool, error) {
	// The tenant clause is here as well as in the policy. `WHERE id = $1` alone
	// would be a schedule id from one organisation editing another's row, and
	// the row-level policy is the layer that catches it — not the only one.
	tag, err := s.db.Exec(s.tenantOf.Bind(ctx, tenantID), `
		UPDATE report_schedules
		   SET report_key = $3, name = $4, params = $5, cron = $6, format = $7,
		       recipients = $8, active = $9, updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID, schedule.ReportKey, schedule.Name, schedule.Params, schedule.Cron,
		schedule.Format, schedule.Recipients, schedule.Active)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) DeleteSchedule(ctx context.Context, tenantID, id string) (string, error) {
	var reportKey string
	err := s.db.QueryRow(s.tenantOf.Bind(ctx, tenantID),
		`DELETE FROM report_schedules WHERE id = $1 AND tenant_id = $2 RETURNING report_key`,
		id, tenantID).Scan(&reportKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", reports.ErrNoSuchSchedule
	}
	return reportKey, err
}

func (s *Store) TenantByRegistration(ctx context.Context, registration string) (string, error) {
	if registration == "" {
		return "", errors.New("a registration number is required")
	}
	var tenantID string
	err := s.db.QueryRow(s.tenantOf.Unbound(ctx),
		`SELECT tenant_id FROM tenant_profiles WHERE registration_number = $1`,
		registration).Scan(&tenantID)
	return tenantID, err
}

func (s *Store) RegistrationOf(ctx context.Context, tenantID string) (string, error) {
	var registration string
	err := s.db.QueryRow(s.tenantOf.Bind(ctx, tenantID),
		`SELECT registration_number FROM tenant_profiles WHERE tenant_id = $1`,
		tenantID).Scan(&registration)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return registration, err
}

func (s *Store) CreateGrant(ctx context.Context, grant reports.Grant) (string, error) {
	var id string
	err := s.db.QueryRow(s.tenantOf.Bind(ctx, grant.GranteeTenantID), `
		INSERT INTO report_grants
		    (grantor_tenant_id, grantee_tenant_id, report_key, scope,
		     counterparty_ref, valid_until, created_by, note)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::uuid, $8)
		RETURNING id`,
		grant.GrantorTenantID, grant.GranteeTenantID, grant.ReportKey, grant.Scope,
		grant.CounterpartyRef, grant.ValidUntil, grant.CreatedBy, grant.Note).Scan(&id)
	// One live agreement per pair per report, held by a partial unique index.
	// Only that violation is the conflict: the handler this was lifted from
	// answered 409 to every failure, so a database that was down told the
	// operator their colleague had already asked.
	if sqlState(err) == "23505" {
		return "", reports.ErrGrantExists
	}
	return id, err
}

func (s *Store) AcceptGrant(ctx context.Context, grantorTenantID, id, actorUserID string) (string, error) {
	// `grantor_tenant_id = $2` is the whole authorization for this statement:
	// the row-level policy lets both parties see the row, so without this
	// clause a grantee could accept their own request.
	var reportKey string
	err := s.db.QueryRow(s.tenantOf.Bind(ctx, grantorTenantID), `
		UPDATE report_grants
		   SET accepted_by = NULLIF($3, '')::uuid, accepted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND grantor_tenant_id = $2 AND revoked_at IS NULL AND accepted_at IS NULL
		 RETURNING report_key`, id, grantorTenantID, actorUserID).Scan(&reportKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", reports.ErrNoPendingRequest
	}
	return reportKey, err
}

func (s *Store) RevokeGrant(ctx context.Context, tenantID, id string) (string, string, error) {
	var reportKey, side string
	err := s.db.QueryRow(s.tenantOf.Bind(ctx, tenantID), `
		UPDATE report_grants
		   SET revoked_at = NOW(), updated_at = NOW()
		 WHERE id = $1
		   AND (grantor_tenant_id = $2 OR grantee_tenant_id = $2)
		   AND revoked_at IS NULL
		 RETURNING report_key,
		           CASE WHEN grantor_tenant_id = $2 THEN 'given' ELSE 'received' END`,
		id, tenantID).Scan(&reportKey, &side)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", reports.ErrNoSuchGrant
	}
	return reportKey, side, err
}

// sqlState is how PostgreSQL says which rule was broken. Anything else — a
// closed connection, a timeout — has no state and is nobody's mistake.
func sqlState(err error) string {
	if err == nil {
		return ""
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState()
	}
	return ""
}

var _ reports.Store = (*Store)(nil)
