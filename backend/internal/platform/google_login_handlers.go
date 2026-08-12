/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Signing in with Google.
 *
 * This is not the federation next door in sso_client_handlers.go, and the
 * difference is the whole design. Federation replaces this deployment's idea of
 * who people are: name a provider and the local sign-in paths close. Google is
 * an *additional* button on this platform's own screen, sitting beside eID and
 * the administrator's password, and it closes nothing.
 *
 * The protocol underneath is identical — Google is an ordinary OpenID Connect
 * provider — so both paths run the same discovery, the same PKCE, the same
 * id_token verification, and land on the same account resolution. What is
 * written twice is only what differs: which cookie the flow parks in, and who
 * is allowed through.
 */

package platform

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/config"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/observability"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/ssoclient"
)

// googleLoginEnabled reports whether this deployment offers the Google button.
func (s *Server) googleLoginEnabled() bool {
	return s.googleLogin != nil && s.googleLogin.Config().Enabled()
}

// googleStartURL is where the sign-in screen sends somebody who presses it.
func (s *Server) googleStartURL() string {
	return config.SelfOrigin() + "/api/v1/auth/google/start"
}

// handleGoogleStart begins a sign-in at Google.
func (s *Server) handleGoogleStart(w http.ResponseWriter, r *http.Request) {
	if !s.googleLoginEnabled() {
		httpx.Error(w, http.StatusNotFound, "Google sign-in is not configured on this deployment")
		return
	}
	// Federation closing the local front door closes this too. Google here is
	// one of *this* platform's ways of establishing who somebody is, and a
	// deployment that has handed that question to a provider must not keep a
	// second answer to it — see requireLocalLogin.
	if !s.localLoginAllowed() {
		httpx.JSON(w, http.StatusForbidden, map[string]any{
			"error":     "this deployment signs in through its SSO provider",
			"code":      "sso_required",
			"start_url": s.ssoStartURL(),
		})
		return
	}

	request, err := s.googleLogin.BeginAuthorization(r.Context())
	if err != nil {
		slog.Error("could not start a sign-in at Google", "error", err)
		s.failGoogle(w, r, "provider_unreachable")
		return
	}

	ssoclient.SetFlowCookie(w, ssoclient.GoogleFlow, ssoclient.Flow{
		State:        request.State,
		Nonce:        request.Nonce,
		CodeVerifier: request.CodeVerifier,
		Next:         ssoclient.SafeNext(r.URL.Query().Get("next"), defaultLandingPath),
	})
	http.Redirect(w, r, request.URL, http.StatusFound)
}

// handleGoogleCallback is where Google returns the browser.
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if !s.googleLoginEnabled() || !s.localLoginAllowed() {
		httpx.Error(w, http.StatusNotFound, "Google sign-in is not configured on this deployment")
		return
	}
	query := r.URL.Query()

	flow, err := ssoclient.ReadFlow(w, r, ssoclient.GoogleFlow, query.Get("state"))
	if err != nil {
		slog.Info("refused a Google callback", "error", err)
		s.failGoogle(w, r, "stale_request")
		return
	}
	if providerError := query.Get("error"); providerError != "" {
		slog.Info("Google refused a sign-in", "error", providerError)
		s.failGoogle(w, r, providerError)
		return
	}
	code := query.Get("code")
	if code == "" {
		s.failGoogle(w, r, "no_code")
		return
	}

	identity, err := s.googleLogin.Exchange(r.Context(), code, flow.CodeVerifier, flow.Nonce)
	if err != nil {
		slog.Error("could not redeem a Google authorization code", "error", err)
		s.failGoogle(w, r, "exchange_failed")
		return
	}

	// Google is asked for the email scope and always returns the claim, so an
	// unverified address here is a real answer rather than a missing one — and
	// it is the answer that matters most, because the address is what an
	// existing local account is matched on. An unverified one would let anybody
	// who can type an address into a Google profile claim somebody else's
	// account here.
	if identity.Email == "" || !identity.EmailVerified {
		slog.Info("refused a Google sign-in with no verified address", "subject", identity.Subject)
		s.failGoogle(w, r, "email_unverified")
		return
	}
	if !ssoclient.EmailInDomains(identity.Email, ssoclient.GoogleAllowedDomains()) {
		slog.Info("refused a Google sign-in from a domain that is not allowed here",
			"domain", domainOf(identity.Email))
		s.failGoogle(w, r, "domain_not_allowed")
		return
	}

	userID, tenantID, err := s.resolveOrProvisionSSOUser(r.Context(), s.googleLogin.Config(), identity)
	if err != nil {
		var refusal signInError
		if errors.As(err, &refusal) {
			slog.Info("refused a verified Google identity", "reason", refusal.Error())
			s.failGoogle(w, r, "no_account")
			return
		}
		slog.Error("could not link a verified Google identity to an account", "error", err)
		s.failGoogle(w, r, "provisioning_failed")
		return
	}

	token, expiresAt, err := s.issueSession(r, userID, tenantID, "google")
	if err != nil {
		slog.Error("could not establish a session for a Google sign-in", "error", err, "user_id", userID)
		s.failGoogle(w, r, "session_failed")
		return
	}
	auth.SetSessionCookie(w, token, expiresAt)
	// Deliberately no id_token cookie. That one exists so signing out here can
	// end the session at the provider this deployment federates to; ending
	// somebody's Google session because they signed out of this platform is not
	// something this platform has any business doing.

	observability.RecordLogin(observability.LoginGoogle, true)
	audit.Record(r.Context(), tenantID, userID, "auth.login_success", "user", map[string]any{
		"method": "google",
		"email":  identity.Email,
	})
	http.Redirect(w, r, config.WebOrigin()+flow.Next, http.StatusFound)
}

// failGoogle returns somebody to the sign-in screen with a reason it can render.
//
// Every refusal on this rail comes through here, which is what makes it the one
// place the failure counter belongs.
func (s *Server) failGoogle(w http.ResponseWriter, r *http.Request, reason string) {
	observability.RecordLogin(observability.LoginGoogle, false)
	http.Redirect(w, r, config.WebOrigin()+"/login?sso_error="+reason, http.StatusFound)
}

// domainOf is for the log line only: the address itself is somebody's, and the
// domain is the part that answers "should this deployment have allowed it".
func domainOf(email string) string {
	if at := strings.LastIndex(email, "@"); at >= 0 {
		return email[at+1:]
	}
	return ""
}
