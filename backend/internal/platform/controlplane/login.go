package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/observability"
	"github.com/jackc/pgx/v5"
)

// Signing in to the console needs two things that are not one thing: something
// known and something held. Neither alone opens it, and the failure of either
// is answered identically, so that an attacker learns nothing about which half
// they got right — or whether the account exists.
const (
	maxLoginBody = 4 << 10
	// maxLoginFailures is lower than the tenant side's five-in-fifteen because
	// the population is different: a handful of operators who know their
	// passwords, not thousands of people with a shared computer.
	maxLoginFailures   = 5
	loginLockoutWindow = 15 * time.Minute
)

// dummyPasswordHash gives a bcrypt comparison to run when the account does not
// exist, so that "no such operator" takes the same time as "wrong password".
// Without it the response time answers a question the response body will not.
var dummyPasswordHash = func() string {
	hash, err := auth.HashPassword("control-plane-missing-account-placeholder")
	if err != nil {
		panic(err)
	}
	return hash
}()

// account is one row of operator_accounts, as sign-in needs it.
type account struct {
	id            string
	email         string
	name          string
	role          Role
	passwordHash  string
	totpSecret    string
	totpConfirmed bool
	totpLastStep  int64
	locked        bool
	disabled      bool
}

// HandleLogin signs an operator in.
//
// The order of the checks is deliberate: the password is verified before the
// code, and a wrong password never says whether the code would have been right.
func (s *Service) HandleLogin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBody)
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		httpx.Error(w, http.StatusBadRequest, "e-mail and password are required")
		return
	}

	ctx := scoped(r.Context())
	acct, found, err := s.lookupAccount(ctx, req.Email)
	if err != nil {
		slog.Error("control plane: could not read the operator account", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "could not sign you in")
		return
	}

	if !found {
		// The comparison is run and discarded. See dummyPasswordHash.
		auth.CheckPasswordHash(req.Password, dummyPasswordHash)
		s.denyLogin(ctx, w, "unknown", acct, false)
		return
	}
	if acct.disabled {
		auth.CheckPasswordHash(req.Password, dummyPasswordHash)
		s.denyLogin(ctx, w, "disabled", acct, true)
		return
	}
	if acct.locked {
		auth.CheckPasswordHash(req.Password, dummyPasswordHash)
		// No failure is counted here: a live lockout that every further attempt
		// extended would let somebody who knows an operator's address keep that
		// operator locked out for as long as they cared to keep typing.
		s.denyLogin(ctx, w, "locked", acct, true)
		return
	}
	if !auth.CheckPasswordHash(req.Password, acct.passwordHash) {
		s.failLogin(ctx, w, "bad_password", acct)
		return
	}

	// An account with no confirmed authenticator cannot sign in at all. That
	// is not a state the bootstrap command leaves behind — it enrols the
	// second factor as it creates the account — and it exists here so that a
	// row inserted by hand, without one, is a row nobody can use rather than a
	// password-only way in.
	if !acct.totpConfirmed || acct.totpSecret == "" {
		slog.Warn("control plane: an operator account has no second factor",
			"operator_id", acct.id, "email", acct.email)
		s.denyLogin(ctx, w, "no_second_factor", acct, true)
		return
	}

	step, ok := verifyTOTP(acct.totpSecret, req.Code, time.Now())
	if !ok || step <= acct.totpLastStep {
		// step <= last is a code that has already been used. It is answered as
		// a wrong code, because to the person typing it that is what it is.
		s.failLogin(ctx, w, "bad_code", acct)
		return
	}

	sess := Session{Operator: Operator{ID: acct.id, Email: acct.email, Name: acct.name, Role: acct.role}}
	var (
		token     string
		expiresAt time.Time
	)
	err = s.do(ctx, sess, Change{
		Action:     "operator.session.begin",
		TargetType: "operator",
		TargetID:   acct.id,
		After:      map[string]any{"role": string(acct.role)},
	}, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`UPDATE operator_accounts
			    SET failed_login_attempts = 0, locked_until = NULL,
			        totp_last_step = $2, last_login_at = NOW(), updated_at = NOW()
			  WHERE id = $1`, acct.id, step); err != nil {
			return err
		}
		var err error
		token, expiresAt, err = s.sessions.create(ctx, tx, acct.id, r.UserAgent(), clientIPFrom(ctx), true)
		return err
	})
	if err != nil {
		slog.Error("control plane: could not complete the sign-in", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "could not sign you in")
		return
	}

	observability.RecordControlPlaneLogin("success")
	SetSessionCookie(w, token, expiresAt)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"operator":   sess.Operator,
		"expires_at": expiresAt,
	})
}

// lookupAccount finds an operator by address, case-insensitively.
func (s *Service) lookupAccount(ctx context.Context, email string) (account, bool, error) {
	var acct account
	var role string
	err := s.db.QueryRow(ctx,
		`SELECT id::text, email, name, role, password_hash, totp_secret,
		        totp_confirmed_at IS NOT NULL, totp_last_step,
		        (locked_until IS NOT NULL AND locked_until > NOW()),
		        disabled_at IS NOT NULL
		   FROM operator_accounts
		  WHERE lower(email) = lower($1)`, email).
		Scan(&acct.id, &acct.email, &acct.name, &role, &acct.passwordHash, &acct.totpSecret,
			&acct.totpConfirmed, &acct.totpLastStep, &acct.locked, &acct.disabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return account{}, false, nil
	}
	if err != nil {
		return account{}, false, err
	}
	acct.role = Role(role)
	return acct, true, nil
}

// failLogin counts a failed attempt, locks the account if it was the last one
// allowed, and answers.
func (s *Service) failLogin(ctx context.Context, w http.ResponseWriter, result string, acct account) {
	// One statement, so that two attempts arriving together cannot both read
	// the same count and write the same number back. A lockout that has run its
	// course starts counting from one rather than continuing — the tenant side
	// explains the reasoning at length in auth_handlers.go, and it is the same
	// reasoning: otherwise one lockout makes every later single mistake lock
	// the account again.
	if _, err := s.db.Exec(ctx,
		`UPDATE operator_accounts
		    SET failed_login_attempts = next.count,
		        locked_until = CASE WHEN next.count >= $2 THEN NOW() + $3::interval END,
		        updated_at = NOW()
		   FROM (
		      SELECT CASE WHEN locked_until IS NOT NULL AND locked_until <= NOW()
		                  THEN 1 ELSE failed_login_attempts + 1 END AS count
		        FROM operator_accounts WHERE id = $1
		   ) AS next
		  WHERE id = $1`,
		acct.id, maxLoginFailures, loginLockoutWindow.String()); err != nil {
		slog.Error("control plane: could not record the failed sign-in", "error", err)
	}
	s.denyLogin(ctx, w, result, acct, true)
}

// denyLogin answers a refused sign-in and leaves a trail.
//
// Every refusal is the same status, the same body and — as far as anything
// outside can tell — the same shape. What differs is what is recorded: the
// metric carries the reason, so a wave of wrong codes against one account looks
// different in Grafana from a wave of unknown addresses, without either being
// visible to whoever is causing it.
//
// The audit row is written for a known account only; there is no operator to
// attribute an unknown address to, and an audit table that accepts rows for
// accounts that do not exist is one an attacker can write to.
func (s *Service) denyLogin(ctx context.Context, w http.ResponseWriter, result string, acct account, known bool) {
	observability.RecordControlPlaneLogin(result)
	if known {
		sess := Session{Operator: Operator{ID: acct.id, Email: acct.email, Name: acct.name, Role: acct.role}}
		// The caller may have hung up; the record of their attempt outlives
		// that, the way the tenant side's audit writes do.
		if err := s.do(context.WithoutCancel(ctx), sess, Change{
			Action:     "operator.session.denied",
			TargetType: "operator",
			TargetID:   acct.id,
			After:      map[string]any{"result": result},
		}, nil); err != nil {
			slog.Warn("control plane: could not record a refused sign-in", "error", err)
		}
	}
	httpx.Error(w, http.StatusUnauthorized, "the e-mail address, password or code was not right")
}

// HandleLogout ends the session the request arrived with.
func (s *Service) HandleLogout(w http.ResponseWriter, r *http.Request) {
	sess, ok := SessionFrom(r.Context())
	if !ok {
		ClearSessionCookie(w)
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "signed out"})
		return
	}

	if err := s.do(r.Context(), sess, Change{
		Action:     "operator.session.end",
		TargetType: "operator",
		TargetID:   sess.ID,
	}, func(ctx context.Context, tx pgx.Tx) error {
		return s.sessions.Revoke(ctx, tx, sess.Token)
	}); err != nil {
		slog.Error("control plane: could not sign the operator out", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "could not sign you out")
		return
	}

	ClearSessionCookie(w)
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "signed out"})
}

// HandleStepUp re-confirms the second factor for the next few minutes.
func (s *Service) HandleStepUp(w http.ResponseWriter, r *http.Request) {
	sess, ok := SessionFrom(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBody)
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		httpx.Error(w, http.StatusBadRequest, "your authenticator code is required")
		return
	}

	ctx := scoped(r.Context())
	acct, found, err := s.lookupAccount(ctx, sess.Email)
	if err != nil {
		slog.Error("control plane: could not read the operator account", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "could not confirm your code")
		return
	}
	step, ok := verifyTOTP(acct.totpSecret, req.Code, time.Now())
	if !found || !ok || step <= acct.totpLastStep {
		observability.RecordControlPlaneLogin("bad_step_up")
		httpx.Error(w, http.StatusUnauthorized, "that code was not right")
		return
	}

	if err := s.do(ctx, sess, Change{
		Action:     "operator.step_up",
		TargetType: "operator",
		TargetID:   sess.ID,
	}, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`UPDATE operator_accounts SET totp_last_step = $2, updated_at = NOW() WHERE id = $1`,
			acct.id, step); err != nil {
			return err
		}
		return s.sessions.MarkSteppedUp(ctx, tx, sess.Token)
	}); err != nil {
		slog.Error("control plane: could not record the step-up", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "could not confirm your code")
		return
	}

	observability.RecordControlPlaneLogin("step_up")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"stepped_up_until": time.Now().Add(StepUpWindow),
	})
}

// HandleMe answers who the caller is, which is what the console's shell asks on
// every page load to decide whether to show itself or the sign-in screen.
func (s *Service) HandleMe(w http.ResponseWriter, r *http.Request) {
	sess, ok := SessionFrom(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"operator":   sess.Operator,
		"expires_at": sess.ExpiresAt,
		"stepped_up": sess.SteppedUp(time.Now()),
	})
}
