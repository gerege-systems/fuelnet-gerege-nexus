/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * The lookup history, read from the audit trail.
 */
package postgres

import (
	"context"
	"encoding/json"

	"github.com/gerege-systems/open-gerege-nexus/backend/domain/egov"
	"github.com/jackc/pgx/v5"
)

// DB is the read this store needs, in pgx's own shape.
type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Store struct{ db DB }

func New(db DB) *Store { return &Store{db: db} }

// Lookups is the last two hundred questions this organisation asked the state.
//
// `xyp.%` is in the query on purpose: events written before the rename are the
// same acts under their old name, and a history that started empty on the day
// this module shipped would look like a history that had been cleared.
func (s *Store) Lookups(ctx context.Context, tenantID string) ([]egov.Lookup, error) {
	rows, err := s.db.Query(ctx,
		`SELECT action, COALESCE(user_id, ''), details, created_at
		   FROM audit_events
		  WHERE tenant_id = $1
		    AND (action LIKE 'egov.%' OR action LIKE 'xyp.%')
		  ORDER BY created_at DESC
		  LIMIT 200`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lookups := make([]egov.Lookup, 0, 32)
	for rows.Next() {
		var lookup egov.Lookup
		var raw []byte
		// A row that will not scan is skipped rather than failing the screen:
		// this is a history, one unreadable entry in it is not worth refusing
		// the other hundred and ninety-nine.
		if err := rows.Scan(&lookup.Action, &lookup.UserID, &raw, &lookup.CreatedAt); err != nil {
			continue
		}
		_ = json.Unmarshal(raw, &lookup.Details)
		lookups = append(lookups, lookup)
	}
	return lookups, nil
}

var _ egov.History = (*Store)(nil)
