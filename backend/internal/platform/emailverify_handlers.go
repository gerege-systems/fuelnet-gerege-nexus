/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package platform provides the core HTTP Server orchestrator, routing table,
 * authentication middleware, and app installer wiring.
 */

package platform

import (
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/config"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/emailverify"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/security"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/tenant"
	"github.com/go-chi/chi/v5"
)

// Email verification.
//
// Two audiences meet on the same endpoint. A platform outside Gerege Nexus
// presents a client key it was issued here; a screen inside the product is
// already carrying a session. Both want the same act performed against the same
// tenant, so they share one endpoint and one audit trail rather than growing a
// second, subtly different implementation for internal callers.

// emailVerifyError maps a service error onto a status.
//
// A rate limit answers with Retry-After, because "too many requests" without a
// number is something an integrator can only respond to by guessing.
func emailVerifyError(w http.ResponseWriter, err error) {
	var invalid *emailverify.InvalidError
	var limited *emailverify.RateLimitedError
	switch {
	case errors.As(err, &invalid):
		writeJSONError(w, http.StatusBadRequest, invalid.Error())
	case errors.As(err, &limited):
		w.Header().Set("Retry-After", strconv.Itoa(int(limited.RetryAfter.Round(time.Second).Seconds())))
		writeJSONError(w, http.StatusTooManyRequests, limited.Error())
	case errors.Is(err, emailverify.ErrUnauthorizedKey):
		writeJSONError(w, http.StatusUnauthorized, "unknown or disabled client key")
	case errors.Is(err, emailverify.ErrClientNotFound):
		writeJSONError(w, http.StatusNotFound, "verification client not found")
	case errors.Is(err, emailverify.ErrDuplicateName):
		writeJSONError(w, http.StatusConflict, err.Error())
	case errors.Is(err, emailverify.ErrLinkSpent):
		writeJSONError(w, http.StatusGone, err.Error())
	case errors.Is(err, emailverify.ErrMailUnavailable):
		writeJSONError(w, http.StatusServiceUnavailable, err.Error())
	default:
		slog.Error("emailverify: request failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "the verification request could not be completed")
	}
}

// clientKeyFromRequest returns a presented verification client key, if the
// Authorization header carries one.
//
// The prefix is what tells it apart from a session bearer token, which arrives
// in the same header. Without that check a client key would be handed to the
// session store, fail to resolve, and the caller would be told their key is an
// expired session.
func clientKeyFromRequest(r *http.Request) string {
	const bearer = "Bearer "
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(header) <= len(bearer) || !strings.EqualFold(header[:len(bearer)], bearer) {
		return ""
	}
	token := strings.TrimSpace(header[len(bearer):])
	if !strings.HasPrefix(token, emailverify.KeyPrefix) {
		return ""
	}
	return token
}

type verifySendRequest struct {
	Email       string `json:"email"`
	RedirectURL string `json:"redirect_url"`
	Purpose     string `json:"purpose"`
	Locale      string `json:"locale"`
}

// handleVerifySend issues a verification link and queues the mail.
//
// It sits outside the authenticated group because the majority of its callers
// are not browsers and hold no session. Authority comes from a client key, and
// falls back to the session for the settings screen's own test send — that
// fallback is what keeps the product from having to call itself over HTTP with
// a key it issued to itself.
func (s *Server) handleVerifySend(w http.ResponseWriter, r *http.Request) {
	var req verifySendRequest
	if decodeLimitedJSON(r, &req, 8<<10) != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid verification request")
		return
	}

	locale := req.Locale
	if locale == "" {
		locale = config.LocaleFromRequest(r)
	}
	base := emailverify.Request{
		Email:       req.Email,
		RedirectURL: req.RedirectURL,
		Purpose:     req.Purpose,
		Locale:      locale,
		ClientIP:    security.ClientIP(r),
	}

	if key := clientKeyFromRequest(r); key != "" {
		client, err := s.emailVerify.Authenticate(r.Context(), key)
		if err != nil {
			emailVerifyError(w, err)
			return
		}
		verification, err := s.emailVerify.SendForClient(r.Context(), client, base)
		if err != nil {
			emailVerifyError(w, err)
			return
		}
		audit.Record(r.Context(), client.TenantID, "client:"+client.ID, "emailverify.send", "email_verification",
			map[string]any{"id": verification.ID, "client": client.Name, "purpose": verification.Purpose})
		writeJSON(w, http.StatusOK, verification)
		return
	}

	// No client key: this must be somebody signed in. The session is resolved
	// here rather than by authMiddleware because the route is shared with the
	// keyed callers above, and mounting it inside the protected group would
	// close it to them.
	claims, err := s.sessions.Resolve(r.Context(), auth.TokenFromRequest(r))
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "a verification client key or an active session is required")
		return
	}
	base.Source = "portal"
	verification, err := s.emailVerify.Send(r.Context(), claims.TenantID, base)
	if err != nil {
		emailVerifyError(w, err)
		return
	}
	audit.Record(r.Context(), claims.TenantID, claims.UserID, "emailverify.send", "email_verification",
		map[string]any{"id": verification.ID, "purpose": verification.Purpose})
	writeJSON(w, http.StatusOK, verification)
}

// handleVerifyConfirm honours the link in the mail.
//
// Unauthenticated by design: the person clicking is the one being verified, and
// they have no account here. The token in the query is the whole authority, it
// is good exactly once, and a spent one is 410 rather than a redirect — sending
// somebody onward on a link that proved nothing is how a stale forward turns
// into an accepted address.
func (s *Server) handleVerifyConfirm(w http.ResponseWriter, r *http.Request) {
	locale := config.LocaleFromRequest(r)
	verification, err := s.emailVerify.Confirm(r.Context(), r.URL.Query().Get("token"))
	if err != nil {
		if !errors.Is(err, emailverify.ErrLinkSpent) {
			slog.Error("emailverify: confirm failed", "error", err)
			writeVerifyPage(w, http.StatusInternalServerError, locale, false)
			return
		}
		writeVerifyPage(w, http.StatusGone, locale, false)
		return
	}

	audit.Record(r.Context(), verification.TenantID, "recipient", "emailverify.confirmed", "email_verification",
		map[string]any{"id": verification.ID, "source": verification.Source, "purpose": verification.Purpose})

	// The destination was validated when the link was issued, against the rules
	// in force then and the client's allowlist. It is not re-derived from
	// anything in this request.
	if verification.RedirectURL != "" {
		http.Redirect(w, r, verification.RedirectURL, http.StatusFound)
		return
	}
	writeVerifyPage(w, http.StatusOK, locale, true)
}

// writeVerifyPage answers a click with a page rather than a JSON body: what
// arrives here is a person in a browser who never asked for an API.
func writeVerifyPage(w http.ResponseWriter, status int, locale string, confirmed bool) {
	title, body := emailverify.ResultPage(locale, confirmed)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	// Both strings are our own, but they are escaped anyway: the day one of
	// them starts carrying a caller's purpose or address, the escaping is
	// already there rather than being the thing somebody forgot.
	_, _ = fmt.Fprintf(w, `<!doctype html>
<html lang="%s"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>body{font-family:system-ui,-apple-system,"Segoe UI",sans-serif;background:#f8fafc;color:#0f172a;display:grid;place-items:center;min-height:100vh;margin:0}
main{background:#fff;border:1px solid #e2e8f0;border-radius:12px;padding:40px;max-width:420px;text-align:center;box-shadow:0 1px 3px rgba(15,23,42,.08)}
h1{font-size:20px;margin:0 0 12px}p{color:#475569;font-size:14px;line-height:1.6;margin:0}</style>
</head><body><main><h1>%s</h1><p>%s</p></main></body></html>`,
		html.EscapeString(locale), html.EscapeString(title), html.EscapeString(title), html.EscapeString(body))
}

// Client administration. Issuing a key that can send mail in the tenant's name
// is an administrative act, so these sit behind requireAdmin with the rest of
// the settings surface.

func (s *Server) handleEmailVerifyOverview(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit := 25
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, convErr := strconv.Atoi(raw); convErr == nil {
			limit = parsed
		}
	}
	overview, err := s.emailVerify.Overview(r.Context(), tenantID, limit)
	if err != nil {
		emailVerifyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (s *Server) handleListEmailVerifyClients(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	clients, err := s.emailVerify.ListClients(r.Context(), tenantID)
	if err != nil {
		emailVerifyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, clients)
}

type emailVerifyClientRequest struct {
	Name                 string   `json:"name"`
	Status               string   `json:"status"`
	HourlyLimit          int      `json:"hourly_limit"`
	AllowedRedirectHosts []string `json:"allowed_redirect_hosts"`
}

func (r emailVerifyClientRequest) toInput() emailverify.ClientInput {
	return emailverify.ClientInput{
		Name:                 r.Name,
		Status:               emailverify.ClientStatus(strings.ToUpper(strings.TrimSpace(r.Status))),
		HourlyLimit:          r.HourlyLimit,
		AllowedRedirectHosts: r.AllowedRedirectHosts,
	}
}

func (s *Server) handleCreateEmailVerifyClient(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req emailVerifyClientRequest
	if decodeLimitedJSON(r, &req, 16<<10) != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid client configuration")
		return
	}
	client, err := s.emailVerify.CreateClient(r.Context(), tenantID, req.toInput())
	if err != nil {
		emailVerifyError(w, err)
		return
	}
	claims, _ := auth.UserFromContext(r.Context())
	audit.Record(r.Context(), tenantID, claims.UserID, "emailverify.client.create", "email_verification_client",
		map[string]any{"id": client.ID, "name": client.Name})
	// The only response that carries the key. There is no endpoint that reads
	// it back, because the database does not have it.
	writeJSON(w, http.StatusCreated, client)
}

func (s *Server) handleUpdateEmailVerifyClient(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req emailVerifyClientRequest
	if decodeLimitedJSON(r, &req, 16<<10) != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid client configuration")
		return
	}
	client, err := s.emailVerify.UpdateClient(r.Context(), tenantID, chi.URLParam(r, "id"), req.toInput())
	if err != nil {
		emailVerifyError(w, err)
		return
	}
	claims, _ := auth.UserFromContext(r.Context())
	audit.Record(r.Context(), tenantID, claims.UserID, "emailverify.client.update", "email_verification_client",
		map[string]any{"id": client.ID, "status": client.Status})
	writeJSON(w, http.StatusOK, client)
}

func (s *Server) handleDeleteEmailVerifyClient(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if err := s.emailVerify.DeleteClient(r.Context(), tenantID, id); err != nil {
		emailVerifyError(w, err)
		return
	}
	claims, _ := auth.UserFromContext(r.Context())
	audit.Record(r.Context(), tenantID, claims.UserID, "emailverify.client.delete", "email_verification_client",
		map[string]any{"id": id})
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
