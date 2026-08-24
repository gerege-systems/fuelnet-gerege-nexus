/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Reading the operator audit trail.
 *
 * Writing it is the console's own middleware, in internal/platform/operator:
 * every change an operator makes is one transaction with its audit row in it,
 * and neither half can be committed without the other. This is the screen that
 * reads what that wrote.
 */

package controlplane

import (
	"context"
	"fmt"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/operator"
)

// ListAudit reads the trail, newest first, optionally narrowed to one action or
// one target.
func (s *Service) ListAudit(ctx context.Context, action, targetType, targetID string) ([]operator.AuditEntry, error) {
	rows, err := s.db.Query(operator.Scoped(ctx),
		`SELECT id::text, operator_id::text, operator_email, action, target_type, target_id,
		        reason, before, after, ip, created_at
		   FROM operator_audit
		  WHERE ($1 = '' OR action = $1)
		    AND ($2 = '' OR target_type = $2)
		    AND ($3 = '' OR target_id = $3)
		  ORDER BY created_at DESC
		  LIMIT $4`,
		action, targetType, targetID, operator.AuditPageSize)
	if err != nil {
		return nil, fmt.Errorf("control plane: read the audit trail: %w", err)
	}
	defer rows.Close()

	entries := make([]operator.AuditEntry, 0, 32)
	for rows.Next() {
		var entry operator.AuditEntry
		if err := rows.Scan(&entry.ID, &entry.OperatorID, &entry.OperatorEmail, &entry.Action,
			&entry.TargetType, &entry.TargetID, &entry.Reason, &entry.Before, &entry.After,
			&entry.IP, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("control plane: read an audit row: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}
