package audit

import (
	"context"
	"log/slog"
	"time"
)

type AuditEvent struct {
	TenantID  string         `json:"tenant_id"`
	UserID    string         `json:"user_id"`
	Action    string         `json:"action"`
	Resource  string         `json:"resource"`
	Details   map[string]any `json:"details"`
	Timestamp time.Time      `json:"timestamp"`
}

func Record(ctx context.Context, tenantID, userID, action, resource string, details map[string]any) {
	slog.Info("AUDIT_EVENT",
		"tenant_id", tenantID,
		"user_id", userID,
		"action", action,
		"resource", resource,
		"details", details,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}
