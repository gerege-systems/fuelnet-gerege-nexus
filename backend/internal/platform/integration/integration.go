/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package integration provides an asynchronous Webhook Event Dispatcher and
 * external system REST Connector Manager with HMAC-SHA256 signature signing.
 */

package integration

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type ConnectorStatus string

const (
	StatusActive   ConnectorStatus = "ACTIVE"
	StatusInactive ConnectorStatus = "INACTIVE"
	StatusError    ConnectorStatus = "ERROR"
)

type IntegrationConfig struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"` // e.g. "webhook", "government", "payment", "custom_rest"
	TargetURL  string            `json:"target_url"`
	SecretKey  string            `json:"secret_key,omitempty"`
	Status     ConnectorStatus   `json:"status"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	LastPingAt time.Time         `json:"last_ping_at"`
}

type EventPayload struct {
	EventID   string         `json:"event_id"`
	EventType string         `json:"event_type"` // e.g. "contact.created", "stock.adjusted", "order.placed"
	TenantID  string         `json:"tenant_id"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

type Manager struct {
	mu           sync.RWMutex
	integrations map[string]*IntegrationConfig
	httpClient   *http.Client
}

func NewManager() *Manager {
	m := &Manager{
		integrations: make(map[string]*IntegrationConfig),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Register default system connectors
	m.Register(&IntegrationConfig{
		ID:         "int_gerege_xyp",
		Name:       "Gerege XYP State Exchange",
		Type:       "government",
		TargetURL:  "https://xyp.gerege.mn/api/v1",
		Status:     StatusActive,
		LastPingAt: time.Now(),
	})

	m.Register(&IntegrationConfig{
		ID:         "int_eid_sso",
		Name:       "Gerege E-ID Digital Identity SSO",
		Type:       "government",
		TargetURL:  "https://eid.gerege.mn/api/v1",
		Status:     StatusActive,
		LastPingAt: time.Now(),
	})

	return m
}

// Register stores a connector. The manager keeps its own copy so that a caller
// still holding the argument cannot mutate registered state without the lock.
func (m *Manager) Register(cfg *IntegrationConfig) {
	stored := *cfg
	if stored.Status == "" {
		stored.Status = StatusActive
	}
	stored.LastPingAt = time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.integrations[stored.ID] = &stored
}

// List returns the registered connectors.
//
// Each entry is a copy with the signing secret cleared. Handing out the stored
// pointers let a caller mutate a live connector without the lock — a data race
// against Register and DispatchEvent — and the secret rode along into the JSON
// of the admin endpoint that renders this. A webhook signing key is written,
// never read back.
func (m *Manager) List() []*IntegrationConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*IntegrationConfig, 0, len(m.integrations))
	for _, cfg := range m.integrations {
		public := *cfg
		public.SecretKey = ""
		list = append(list, &public)
	}
	return list
}

// dispatchTimeout bounds one webhook delivery. It exists because the delivery
// no longer inherits the caller's cancellation (see DispatchEvent) and so needs
// a deadline of its own.
const dispatchTimeout = 15 * time.Second

func (m *Manager) DispatchEvent(ctx context.Context, payload EventPayload) error {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// Snapshot the subscribers under the lock, then dispatch outside it: the
	// goroutines outlive this call, and holding a read lock for their whole
	// lifetime would block every Register behind the slowest subscriber.
	type target struct{ url, secret string }
	m.mu.RLock()
	targets := make([]target, 0, len(m.integrations))
	for _, cfg := range m.integrations {
		if cfg.Status != StatusActive || cfg.TargetURL == "" || cfg.Type != "webhook" {
			continue
		}
		targets = append(targets, target{url: cfg.TargetURL, secret: cfg.SecretKey})
	}
	m.mu.RUnlock()

	// Delivery is asynchronous, so it must not carry the caller's cancellation.
	// ctx is typically a request context: it is cancelled the moment the handler
	// that called DispatchEvent returns, which is almost always before the POST
	// has left the process — so every webhook raced its own cancellation and
	// most were never delivered. WithoutCancel keeps the trace and request
	// values that make the outbound call attributable, and drops only the
	// cancellation; the timeout above supplies the deadline instead.
	sendCtx := context.WithoutCancel(ctx)

	for _, t := range targets {
		go func(targetURL, secret string) {
			reqCtx, cancel := context.WithTimeout(sendCtx, dispatchTimeout)
			defer cancel()

			req, err := http.NewRequestWithContext(reqCtx, "POST", targetURL, bytes.NewReader(bodyBytes))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			// X-ERP-* are legacy compatibility header names, kept through the
			// Gerege Nexus rebrand. Subscribers read these exact names and
			// verify the signature against them; renaming would break every
			// existing endpoint silently. A Nexus-named alias would need
			// dual-emission and a deprecation window, not a rename.
			req.Header.Set("X-ERP-Event", payload.EventType)

			if secret != "" {
				mac := hmac.New(sha256.New, []byte(secret))
				mac.Write(bodyBytes)
				sig := hex.EncodeToString(mac.Sum(nil))
				req.Header.Set("X-ERP-Signature", sig)
			}

			resp, err := m.httpClient.Do(req)
			if err == nil {
				// Drain before closing so the connection returns to the pool
				// instead of being torn down after every event.
				_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
				_ = resp.Body.Close()
			}
		}(t.url, t.secret)
	}

	return nil
}
