package platform

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"golang.org/x/crypto/chacha20poly1305"
)

var validStaffPIN = regexp.MustCompile(`^[0-9]{4,12}$`)

func (s *Server) handleSetStaffPIN(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.UserFromContext(r.Context())
	var req struct {
		MembershipID string `json:"membership_id"`
		PIN          string `json:"pin"`
	}
	if decodeLimitedJSON(r, &req, 8<<10) != nil || !validStaffPIN.MatchString(req.PIN) {
		httpx.Error(w, 400, "PIN must contain 4-12 digits")
		return
	}
	hash, err := auth.HashPassword(req.PIN)
	if err != nil {
		httpx.Error(w, 500, "failed to protect PIN")
		return
	}
	result, err := s.db.Exec(r.Context(), `INSERT INTO staff_pin_credentials(membership_id,tenant_id,pin_hash) SELECT id,tenant_id,$3 FROM memberships WHERE id=$1 AND tenant_id=$2 ON CONFLICT(membership_id) DO UPDATE SET pin_hash=EXCLUDED.pin_hash,active=true,failed_attempts=0,locked_until=NULL,updated_at=NOW()`, req.MembershipID, claims.TenantID, hash)
	if err != nil || result.RowsAffected() != 1 {
		httpx.Error(w, 404, "membership not found")
		return
	}
	audit.Record(r.Context(), claims.TenantID, claims.UserID, "staff.pin_changed", "membership", map[string]any{"membership_id": req.MembershipID})
	httpx.JSON(w, 200, map[string]string{"status": "updated"})
}

func (s *Server) handleDeviceStaffPIN(w http.ResponseWriter, r *http.Request) {
	device := r.Context().Value(deviceContextKey{}).(deviceClaims)
	if device.FormFactor != "pos" && device.FormFactor != "tablet" {
		httpx.Error(w, 403, "staff switching is unavailable on this device")
		return
	}
	var req struct {
		PIN string `json:"pin"`
	}
	if decodeLimitedJSON(r, &req, 4<<10) != nil || !validStaffPIN.MatchString(req.PIN) {
		httpx.Error(w, 401, "invalid PIN")
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT p.membership_id::text,p.pin_hash,m.user_id::text,u.name,u.email,p.locked_until FROM staff_pin_credentials p JOIN memberships m ON m.id=p.membership_id JOIN users u ON u.id=m.user_id WHERE p.tenant_id=$1 AND p.active`, device.TenantID)
	if err != nil {
		httpx.Error(w, 503, "staff authentication unavailable")
		return
	}
	defer rows.Close()
	var membershipID, userID, name, email string
	matched := false
	for rows.Next() {
		var mid, hash, uid, n, e string
		var locked *time.Time
		if rows.Scan(&mid, &hash, &uid, &n, &e, &locked) == nil && (locked == nil || locked.Before(time.Now())) && auth.CheckPasswordHash(req.PIN, hash) {
			membershipID, userID, name, email, matched = mid, uid, n, e, true
			break
		}
	}
	if !matched {
		audit.Record(r.Context(), device.TenantID, "device:"+device.ID, "staff.pin_failed", "device", nil)
		httpx.Error(w, 401, "invalid PIN")
		return
	}
	token, expires, err := s.issueSession(r, userID, device.TenantID, "staff-pin")
	if err != nil {
		httpx.Error(w, 500, "failed to establish staff session")
		return
	}
	auth.SetSessionCookie(w, token, expires)
	audit.Record(r.Context(), device.TenantID, userID, "staff.pin_success", "device", map[string]any{"device_id": device.ID})
	httpx.JSON(w, 200, map[string]any{"expires_at": expires, "membership_id": membershipID, "user": map[string]any{"id": userID, "name": name, "email": email, "tenant_id": device.TenantID}})
}

func (s *Server) handleOpenShift(w http.ResponseWriter, r *http.Request) {
	device := r.Context().Value(deviceContextKey{}).(deviceClaims)
	user, _ := auth.UserFromContext(r.Context())
	var req struct {
		OpeningFloat float64 `json:"opening_float"`
		Notes        string  `json:"notes"`
	}
	if decodeLimitedJSON(r, &req, 8<<10) != nil || req.OpeningFloat < 0 {
		httpx.Error(w, 400, "invalid shift")
		return
	}
	var membershipID, shiftID string
	err := s.db.QueryRow(r.Context(), `WITH member AS (SELECT id FROM memberships WHERE tenant_id=$1 AND user_id=$2 LIMIT 1) INSERT INTO pos_shifts(tenant_id,device_id,membership_id,opening_float,notes) SELECT $1,$3,id,$4,$5 FROM member RETURNING id::text`, user.TenantID, user.UserID, device.ID, req.OpeningFloat, strings.TrimSpace(req.Notes)).Scan(&shiftID)
	_ = membershipID
	if err != nil {
		httpx.Error(w, 409, "device already has an open shift")
		return
	}
	audit.Record(r.Context(), user.TenantID, user.UserID, "shift.opened", "shift", map[string]any{"shift_id": shiftID, "device_id": device.ID})
	httpx.JSON(w, 201, map[string]any{"id": shiftID, "opened_at": time.Now()})
}

func (s *Server) handleCloseShift(w http.ResponseWriter, r *http.Request) {
	device := r.Context().Value(deviceContextKey{}).(deviceClaims)
	user, _ := auth.UserFromContext(r.Context())
	var req struct {
		ClosingTotal float64 `json:"closing_total"`
		Notes        string  `json:"notes"`
	}
	if decodeLimitedJSON(r, &req, 8<<10) != nil || req.ClosingTotal < 0 {
		httpx.Error(w, 400, "invalid shift")
		return
	}
	var id string
	err := s.db.QueryRow(r.Context(), `UPDATE pos_shifts SET closed_at=NOW(),closing_total=$3,notes=CASE WHEN $4='' THEN notes ELSE $4 END WHERE tenant_id=$1 AND device_id=$2 AND closed_at IS NULL RETURNING id::text`, user.TenantID, device.ID, req.ClosingTotal, strings.TrimSpace(req.Notes)).Scan(&id)
	if err != nil {
		httpx.Error(w, 404, "no open shift")
		return
	}
	audit.Record(r.Context(), user.TenantID, user.UserID, "shift.closed", "shift", map[string]any{"shift_id": id})
	httpx.JSON(w, 200, map[string]any{"id": id, "status": "CLOSED"})
}

func (s *Server) handleCurrentShift(w http.ResponseWriter, r *http.Request) {
	device := r.Context().Value(deviceContextKey{}).(deviceClaims)
	var id, membershipID string
	var opened time.Time
	var opening float64
	err := s.db.QueryRow(r.Context(), `SELECT id::text,membership_id::text,opened_at,opening_float FROM pos_shifts WHERE tenant_id=$1 AND device_id=$2 AND closed_at IS NULL`, device.TenantID, device.ID).Scan(&id, &membershipID, &opened, &opening)
	if err != nil {
		httpx.JSON(w, 200, map[string]any{"shift": nil})
		return
	}
	httpx.JSON(w, 200, map[string]any{"shift": map[string]any{"id": id, "membership_id": membershipID, "opened_at": opened, "opening_float": opening}})
}

type telemetryEvent struct {
	Level, Event string
	Payload      map[string]any
	OccurredAt   time.Time `json:"occurred_at"`
}

func (s *Server) handleDeviceTelemetry(w http.ResponseWriter, r *http.Request) {
	device := r.Context().Value(deviceContextKey{}).(deviceClaims)
	var req struct {
		Events []telemetryEvent `json:"events"`
	}
	if decodeLimitedJSON(r, &req, 256<<10) != nil || len(req.Events) == 0 || len(req.Events) > 100 {
		httpx.Error(w, 400, "invalid telemetry batch")
		return
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		httpx.Error(w, 503, "telemetry unavailable")
		return
	}
	defer tx.Rollback(r.Context())
	for _, event := range req.Events {
		level := strings.ToUpper(event.Level)
		if level != "INFO" && level != "WARN" && level != "ERROR" || strings.TrimSpace(event.Event) == "" {
			httpx.Error(w, 400, "invalid telemetry event")
			return
		}
		if event.OccurredAt.IsZero() {
			event.OccurredAt = time.Now()
		}
		raw, _ := json.Marshal(event.Payload)
		if _, err = tx.Exec(r.Context(), `INSERT INTO device_telemetry(tenant_id,device_id,level,event,payload,occurred_at) VALUES($1,$2,$3,$4,$5,$6)`, device.TenantID, device.ID, level, event.Event, raw, event.OccurredAt); err != nil {
			httpx.Error(w, 503, "telemetry unavailable")
			return
		}
	}
	if tx.Commit(r.Context()) != nil {
		httpx.Error(w, 503, "telemetry unavailable")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func encryptPushToken(token string) (string, error) {
	keyText := os.Getenv("PUSH_TOKEN_ENCRYPTION_KEY")
	if keyText == "" {
		return "", errors.New("push token encryption key is missing")
	}
	key, err := base64.StdEncoding.DecodeString(keyText)
	if err != nil || len(key) != chacha20poly1305.KeySize {
		return "", errors.New("push token encryption key is invalid")
	}
	aead, _ := chacha20poly1305.NewX(key)
	nonce := make([]byte, aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(append(nonce, aead.Seal(nil, nonce, []byte(token), nil)...)), nil
}

func (s *Server) handleRegisterPushToken(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.UserFromContext(r.Context())
	var req struct {
		Token    string `json:"token"`
		Provider string `json:"provider"`
		AppID    string `json:"app_id"`
	}
	if decodeLimitedJSON(r, &req, 16<<10) != nil || len(req.Token) < 16 {
		httpx.Error(w, 400, "invalid push token")
		return
	}
	req.Provider = strings.ToUpper(req.Provider)
	if req.Provider != "APNS" && req.Provider != "FCM" {
		httpx.Error(w, 400, "invalid push provider")
		return
	}
	encrypted, err := encryptPushToken(req.Token)
	if err != nil {
		httpx.Error(w, 503, "push registration is not configured")
		return
	}
	_, err = s.db.Exec(r.Context(), `INSERT INTO push_tokens(tenant_id,user_id,provider,token_hash,token_ciphertext,app_id) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(token_hash) DO UPDATE SET user_id=EXCLUDED.user_id,tenant_id=EXCLUDED.tenant_id,provider=EXCLUDED.provider,token_ciphertext=EXCLUDED.token_ciphertext,app_id=EXCLUDED.app_id,updated_at=NOW()`, claims.TenantID, claims.UserID, req.Provider, caseSensitiveSecretHash(req.Token), encrypted, req.AppID)
	if err != nil {
		httpx.Error(w, 503, "push registration failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
