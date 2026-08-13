package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/security"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/time/rate"
)

// loginRate is what one address may spend on sign-in attempts: a burst of five,
// refilled at one every twelve seconds. nginx has its own limit in front of
// this (deploy/nginx/cp.nexus.gerege.mn.conf) and that is the one that matters
// under load; this is the layer that still applies when the console is reached
// some other way — a second proxy, a port-forward during an incident.
var loginRate = rate.Every(12e9)

// Routes mounts the console's API. Called by the platform's route table as
// r.Route("/cp/api", service.Routes), so that everything in here shares the
// same three guarantees and no route can be added that quietly skips one.
func (s *Service) Routes(r chi.Router) {
	// Order matters and is the whole design.
	//
	// HostGate first: a request to the wrong hostname is answered before it
	// touches a database or a rate limiter, so the console costs nothing to
	// the traffic that is not for it.
	//
	// RequireAudit next, above authentication rather than below it, so that it
	// covers the sign-in route too — beginning a session is a write, and it is
	// one of the writes an operator audit exists to hold.
	r.Use(s.HostGate)
	r.Use(s.RequireAudit)

	r.Group(func(anon chi.Router) {
		anon.Use(security.RateLimitMiddleware(security.NewIPRateLimiter(loginRate, 5)))
		anon.Post("/session", s.HandleLogin)
	})

	r.Group(func(signedIn chi.Router) {
		signedIn.Use(s.RequireOperator)

		signedIn.Get("/me", s.HandleMe)
		signedIn.Delete("/session", s.HandleLogout)
		signedIn.Post("/step-up", s.HandleStepUp)

		signedIn.With(s.RequireCapability(CapTenantRead)).Get("/tenants", s.handleListTenants)
		signedIn.With(s.RequireCapability(CapTenantRead)).Get("/tenants/{id}", s.handleGetTenant)
		signedIn.With(s.RequireCapability(CapAuditRead)).Get("/audit", s.handleListAudit)
		signedIn.With(s.RequireCapability(CapOperatorRead)).Get("/operators", s.handleListOperators)

		// The organisation's life. Suspension is reversible and needs one
		// operator; deletion is not and needs two, which is why it goes
		// through the approvals below rather than having a route of its own
		// that does the deed.
		signedIn.With(s.RequireCapability(CapTenantCreate)).
			Post("/tenants", s.handleCreateTenant)
		signedIn.With(s.RequireCapability(CapTenantSuspend), s.RequireStepUp).
			Post("/tenants/{id}/suspend", s.handleSuspendTenant)
		signedIn.With(s.RequireCapability(CapTenantSuspend), s.RequireStepUp).
			Post("/tenants/{id}/resume", s.handleResumeTenant)
		signedIn.With(s.RequireCapability(CapTenantDelete), s.RequireStepUp).
			Post("/tenants/{id}/deletion", s.handleRequestDeletion)
		// Cancelling a deletion needs neither a second person nor a second
		// factor. It is the safe direction: the asymmetry is the point of a
		// grace period, and a recovery that is harder than the mistake is a
		// recovery nobody manages in time.
		signedIn.With(s.RequireCapability(CapTenantSuspend)).
			Delete("/tenants/{id}/deletion", s.handleCancelDeletion)
		// The export reads the organisation's actual data, so it is gated like
		// the deletion it usually precedes rather than like a read: the same
		// capability, and a second factor. See export.go for why this one
		// action is allowed to leave the console's usual boundary.
		signedIn.With(s.RequireCapability(CapTenantDelete), s.RequireStepUp).
			Get("/tenants/{id}/export", s.handleExportTenant)
		signedIn.With(s.RequireCapability(CapQuotaWrite), s.RequireStepUp).
			Put("/tenants/{id}/quota", s.handleSetQuota)

		// What is counting down. On its own route rather than inside the
		// organisation list, because it is the one screen an operator should
		// look at without being asked to.
		signedIn.With(s.RequireCapability(CapTenantRead)).Get("/deletions", s.handleListDeletions)

		signedIn.With(s.RequireCapability(CapApprove)).Get("/approvals", s.handleListApprovals)
		signedIn.With(s.RequireCapability(CapApprove), s.RequireStepUp).
			Post("/approvals/{id}/approve", s.handleApprove)
		signedIn.With(s.RequireCapability(CapApprove)).
			Post("/approvals/{id}/reject", s.handleReject)

		// The help desk.
		signedIn.With(s.RequireCapability(CapSupport)).Get("/people", s.handleFindPeople)
		signedIn.With(s.RequireCapability(CapSupport), s.RequireStepUp).
			Post("/people/{id}/unlock", s.handleUnlock)
		signedIn.With(s.RequireCapability(CapSupport), s.RequireStepUp).
			Post("/people/{id}/sessions/revoke", s.handleRevokeSessions)
		signedIn.With(s.RequireCapability(CapSupport), s.RequireStepUp).
			Post("/people/{id}/credential-link", s.handleCredentialLink)

		signedIn.With(s.RequireCapability(CapImpersonate), s.RequireStepUp).
			Post("/tenants/{id}/impersonate", s.handleImpersonate)

		// How the platform behaves. Reading is part of the tenant-read
		// capability because "what is this deployment configured to do" is
		// context for every other screen; writing is its own.
		signedIn.With(s.RequireCapability(CapTenantRead)).Get("/settings", s.handleListSettings)
		signedIn.With(s.RequireCapability(CapTenantRead)).Get("/settings/history", s.handleSettingHistory)
		// Step-up on the write: the access mode is here, and switching a
		// platform to public is the single most consequential field in the
		// console.
		signedIn.With(s.RequireCapability(CapSettingsWrite), s.RequireStepUp).
			Put("/settings/{key}", s.handleSetSetting)
		signedIn.With(s.RequireCapability(CapSettingsWrite), s.RequireStepUp).
			Post("/settings/rollback/{id}", s.handleRollbackSetting)

		signedIn.With(s.RequireCapability(CapTenantRead)).Get("/flags", s.handleListFlags)
		signedIn.With(s.RequireCapability(CapFlagsWrite)).Post("/flags", s.handleSaveFlag)
		signedIn.With(s.RequireCapability(CapFlagsWrite)).Delete("/flags/{key}", s.handleDeleteFlag)
		signedIn.With(s.RequireCapability(CapFlagsWrite)).
			Put("/flags/{key}/override", s.handleFlagOverride)

		signedIn.With(s.RequireCapability(CapSettingsWrite)).
			Post("/tenants/{id}/maintenance", s.handleTenantMaintenance)

		// What each organisation used, and the same thing as a spreadsheet.
		signedIn.With(s.RequireCapability(CapTenantRead)).
			Get("/tenants/{id}/usage", s.handleUsage)
		signedIn.With(s.RequireCapability(CapTenantRead)).
			Get("/tenants/{id}/usage.csv", s.handleUsageCSV)

		// The front page, and the operations behind it.
		signedIn.With(s.RequireCapability(CapTenantRead)).Get("/health", s.handleHealth)
		signedIn.With(s.RequireCapability(CapTenantRead)).Get("/catalog/status", s.handleCatalogStatusRoute)
		signedIn.With(s.RequireCapability(CapTenantRead)).Get("/catalog/overview", s.handleCatalogOverviewRoute)
		signedIn.With(s.RequireCapability(CapSettingsWrite), s.RequireStepUp).
			Post("/catalog/sync", s.handleCatalogSyncRoute)
		signedIn.With(s.RequireCapability(CapDeploy), s.RequireStepUp).
			Post("/deploy", s.handleDeploy)
		signedIn.With(s.RequireCapability(CapSettingsWrite)).
			Post("/backups/restore-test", s.handleRestoreTest)

		signedIn.With(s.RequireCapability(CapTenantRead)).Get("/announcements", s.handleListAnnouncements)
		signedIn.With(s.RequireCapability(CapAnnounce)).Post("/announcements", s.handleAnnounce)
		signedIn.With(s.RequireCapability(CapAnnounce)).
			Delete("/announcements/{id}", s.handleWithdrawAnnouncement)
	})
}

// reasoned is the body every write on this console carries. A reason is not
// optional and not defaulted: Do refuses without one, so a handler that forgot
// to read it fails on the first attempt rather than filling the audit trail
// with empty strings.
type reasoned struct {
	Reason string `json:"reason"`
}

// decode reads a JSON body, bounded. Returns false having already answered.
func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxWriteBody)
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		httpx.Error(w, http.StatusBadRequest, "the request body could not be read")
		return false
	}
	return true
}

// maxWriteBody bounds a console write. Generous for a form, small enough that
// nothing here is a way to make the process allocate.
const maxWriteBody = 32 << 10

// fail turns the package's sentinel errors into the answers they deserve, so
// that every handler below is three lines rather than fifteen.
func fail(w http.ResponseWriter, err error, doing string) {
	switch {
	case errors.Is(err, ErrTenantNotFound), errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrApprovalNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrReasonRequired), errors.Is(err, ErrInvalidSlug),
		errors.Is(err, ErrSlugTaken), errors.Is(err, ErrNotSuspended),
		errors.Is(err, ErrNotScheduled), errors.Is(err, ErrAlreadyScheduled),
		errors.Is(err, ErrNotAMember), errors.Is(err, ErrTenantSuspended),
		errors.Is(err, ErrUnknownEnforcement), errors.Is(err, ErrMailNotConfigured),
		errors.Is(err, ErrDeployNotConfigured), errors.Is(err, ErrDeployRefused),
		errors.Is(err, ErrHistoryNotFound), errors.Is(err, ErrNoSettingsStore),
		errors.Is(err, ErrNoFlagStore):
		// Refusals the operator can act on, in words they can act on. These
		// are the platform's own sentinels, never a database error's text —
		// see the default.
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrSelfApproval):
		httpx.Error(w, http.StatusForbidden, err.Error())
	default:
		// Anything else is logged in full and answered vaguely: an error from
		// PostgreSQL describes the schema, and the console is not the place to
		// publish it.
		slog.Error("control plane: "+doing, "error", err)
		httpx.Error(w, http.StatusInternalServerError, "that could not be completed")
	}
}

func (s *Service) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var params NewTenant
	if !decode(w, r, &params) {
		return
	}
	created, err := s.CreateTenant(r.Context(), sess, params)
	if err != nil {
		fail(w, err, "could not create the organisation")
		return
	}
	httpx.JSON(w, http.StatusCreated, created)
}

func (s *Service) handleSuspendTenant(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body reasoned
	if !decode(w, r, &body) {
		return
	}
	if err := s.Suspend(r.Context(), sess, chi.URLParam(r, "id"), body.Reason); err != nil {
		fail(w, err, "could not suspend the organisation")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "suspended"})
}

func (s *Service) handleResumeTenant(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body reasoned
	if !decode(w, r, &body) {
		return
	}
	if err := s.Resume(r.Context(), sess, chi.URLParam(r, "id"), body.Reason); err != nil {
		fail(w, err, "could not resume the organisation")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "active"})
}

func (s *Service) handleRequestDeletion(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body reasoned
	if !decode(w, r, &body) {
		return
	}
	approvalID, err := s.RequestDeletion(r.Context(), sess, chi.URLParam(r, "id"), body.Reason)
	if err != nil {
		fail(w, err, "could not ask for the organisation to be deleted")
		return
	}
	// Deliberately explicit about what has and has not happened: the operator
	// pressed "delete" and nothing has been deleted.
	httpx.JSON(w, http.StatusAccepted, map[string]any{
		"status":      "awaiting a second superadmin",
		"approval_id": approvalID,
		"grace_days":  int(DeletionGrace.Hours() / 24),
	})
}

func (s *Service) handleCancelDeletion(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body reasoned
	if !decode(w, r, &body) {
		return
	}
	if err := s.CancelDeletion(r.Context(), sess, chi.URLParam(r, "id"), body.Reason); err != nil {
		fail(w, err, "could not cancel the deletion")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deletion cancelled"})
}

func (s *Service) handleExportTenant(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	bundle, err := s.ExportTenant(r.Context(), sess, chi.URLParam(r, "id"))
	if err != nil {
		fail(w, err, "could not export the organisation")
		return
	}
	w.Header().Set("Content-Disposition",
		`attachment; filename="`+bundle.Tenant.Slug+`-export.json"`)
	httpx.JSON(w, http.StatusOK, bundle)
}

func (s *Service) handleSetQuota(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body struct {
		Quota
		Reason string `json:"reason"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := s.SetQuota(r.Context(), sess, chi.URLParam(r, "id"), body.Quota, body.Reason); err != nil {
		fail(w, err, "could not set the limits")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (s *Service) handleListDeletions(w http.ResponseWriter, r *http.Request) {
	pending, err := s.TenantsAwaitingDeletion(r.Context())
	if err != nil {
		fail(w, err, "could not list the organisations awaiting deletion")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"tenants": pending})
}

func (s *Service) handleListApprovals(w http.ResponseWriter, r *http.Request) {
	approvals, err := s.ListApprovals(r.Context())
	if err != nil {
		fail(w, err, "could not list the open requests")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"approvals": approvals})
}

func (s *Service) handleApprove(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body reasoned
	if !decode(w, r, &body) {
		return
	}
	if err := s.Approve(r.Context(), sess, chi.URLParam(r, "id"), body.Reason); err != nil {
		fail(w, err, "could not approve the request")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (s *Service) handleReject(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body reasoned
	if !decode(w, r, &body) {
		return
	}
	if err := s.Reject(r.Context(), sess, chi.URLParam(r, "id"), body.Reason); err != nil {
		fail(w, err, "could not reject the request")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func (s *Service) handleFindPeople(w http.ResponseWriter, r *http.Request) {
	people, err := s.FindPeople(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		fail(w, err, "could not search for people")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"people": people})
}

func (s *Service) handleUnlock(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body reasoned
	if !decode(w, r, &body) {
		return
	}
	if err := s.Unlock(r.Context(), sess, chi.URLParam(r, "id"), body.Reason); err != nil {
		fail(w, err, "could not unlock the account")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "unlocked"})
}

func (s *Service) handleRevokeSessions(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body reasoned
	if !decode(w, r, &body) {
		return
	}
	ended, err := s.RevokeSessions(r.Context(), sess, chi.URLParam(r, "id"), body.Reason)
	if err != nil {
		fail(w, err, "could not end the sessions")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": "revoked", "sessions": ended})
}

func (s *Service) handleCredentialLink(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body struct {
		reasoned
		// TenantID is which organisation the mail is sent on behalf of. The
		// verification service counts its quota per organisation, so the
		// answer cannot be "none" — the console sends the one the operator was
		// looking at.
		TenantID string `json:"tenant_id"`
		Purpose  string `json:"purpose"`
	}
	if !decode(w, r, &body) {
		return
	}
	if body.Purpose == "" {
		body.Purpose = "reset"
	}
	if err := s.SendCredentialLink(r.Context(), sess, chi.URLParam(r, "id"),
		body.TenantID, body.Purpose, body.Reason); err != nil {
		fail(w, err, "could not send the link")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Service) handleImpersonate(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body struct {
		reasoned
		UserID string `json:"user_id"`
	}
	if !decode(w, r, &body) {
		return
	}
	link, err := s.BeginImpersonation(r.Context(), sess, chi.URLParam(r, "id"), body.UserID, body.Reason)
	if err != nil {
		fail(w, err, "could not start the session")
		return
	}
	// The link is returned rather than redirected to: the console is on
	// another hostname, and the operator's browser has to make the journey
	// itself for the cookie to land where it belongs.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"url":     link,
		"minutes": int(ImpersonationWindow.Minutes()),
	})
}

func (s *Service) handleListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := s.ListTenants(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		slog.Error("control plane: could not list the organisations", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "could not read the organisations")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"tenants": tenants})
}

func (s *Service) handleGetTenant(w http.ResponseWriter, r *http.Request) {
	detail, err := s.GetTenant(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, ErrTenantNotFound) {
		httpx.Error(w, http.StatusNotFound, "no such organisation")
		return
	}
	if err != nil {
		// An id that is not a UUID reaches here as a database error rather than
		// as ErrTenantNotFound, and answering 500 for a typed-in URL would put
		// a red line in the error tracker for somebody's slip.
		if isInvalidUUID(err) {
			httpx.Error(w, http.StatusNotFound, "no such organisation")
			return
		}
		slog.Error("control plane: could not read the organisation", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "could not read the organisation")
		return
	}
	httpx.JSON(w, http.StatusOK, detail)
}

func (s *Service) handleListAudit(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	entries, err := s.ListAudit(r.Context(),
		query.Get("action"), query.Get("target_type"), query.Get("target_id"))
	if err != nil {
		slog.Error("control plane: could not read the audit trail", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "could not read the audit trail")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"entries": entries})
}

func (s *Service) handleListOperators(w http.ResponseWriter, r *http.Request) {
	operators, err := s.ListOperators(r.Context())
	if err != nil {
		slog.Error("control plane: could not list the operators", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "could not read the operators")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"operators": operators})
}

// The configuration screens.

func (s *Service) handleListSettings(w http.ResponseWriter, r *http.Request) {
	values, err := s.ListSettings(r.Context())
	if err != nil {
		fail(w, err, "could not read the settings")
		return
	}
	// The warnings ride along with the values, because they are about the
	// values: a configuration that contradicts itself belongs on the screen
	// where somebody can fix it.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"settings": values,
		"warnings": s.warnings(),
	})
}

func (s *Service) handleSettingHistory(w http.ResponseWriter, r *http.Request) {
	changes, err := s.SettingHistory(r.Context(), r.URL.Query().Get("key"))
	if err != nil {
		fail(w, err, "could not read the history")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"changes": changes})
}

func (s *Service) handleSetSetting(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body struct {
		reasoned
		Value string `json:"value"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := s.SetSetting(r.Context(), sess, chi.URLParam(r, "key"), body.Value, body.Reason); err != nil {
		fail(w, err, "could not change the setting")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (s *Service) handleRollbackSetting(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body reasoned
	if !decode(w, r, &body) {
		return
	}
	if err := s.RollbackSetting(r.Context(), sess, chi.URLParam(r, "id"), body.Reason); err != nil {
		fail(w, err, "could not roll the setting back")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "rolled back"})
}

func (s *Service) handleListFlags(w http.ResponseWriter, r *http.Request) {
	list, err := s.ListFlags(r.Context())
	if err != nil {
		fail(w, err, "could not read the flags")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"flags": list})
}

func (s *Service) handleSaveFlag(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var input FlagInput
	if !decode(w, r, &input) {
		return
	}
	if err := s.SaveFlag(r.Context(), sess, input); err != nil {
		fail(w, err, "could not save the flag")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (s *Service) handleDeleteFlag(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body reasoned
	if !decode(w, r, &body) {
		return
	}
	if err := s.DeleteFlag(r.Context(), sess, chi.URLParam(r, "key"), body.Reason); err != nil {
		fail(w, err, "could not delete the flag")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Service) handleFlagOverride(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body struct {
		reasoned
		TenantID string `json:"tenant_id"`
		// A pointer: null means "remove the override and go back to the
		// rollout", which is a different instruction from "off".
		Enabled *bool `json:"enabled"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := s.SetFlagOverride(r.Context(), sess, chi.URLParam(r, "key"),
		body.TenantID, body.Enabled, body.Reason); err != nil {
		fail(w, err, "could not set the override")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (s *Service) handleTenantMaintenance(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body struct {
		reasoned
		On      bool   `json:"on"`
		Message string `json:"message"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := s.SetTenantMaintenance(r.Context(), sess, chi.URLParam(r, "id"),
		body.On, body.Message, body.Reason); err != nil {
		fail(w, err, "could not change the maintenance state")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (s *Service) handleListAnnouncements(w http.ResponseWriter, r *http.Request) {
	announcements, err := s.ListAnnouncements(r.Context())
	if err != nil {
		fail(w, err, "could not read the announcements")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"announcements": announcements})
}

func (s *Service) handleAnnounce(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body struct {
		Announcement
		Reason string `json:"reason"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := s.Announce(r.Context(), sess, body.Announcement, body.Reason); err != nil {
		fail(w, err, "could not publish the announcement")
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]string{"status": "published"})
}

func (s *Service) handleWithdrawAnnouncement(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body reasoned
	if !decode(w, r, &body) {
		return
	}
	if err := s.WithdrawAnnouncement(r.Context(), sess, chi.URLParam(r, "id"), body.Reason); err != nil {
		fail(w, err, "could not withdraw the announcement")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "withdrawn"})
}

// The front page and the operations behind it.

func (s *Service) handleHealth(w http.ResponseWriter, r *http.Request) {
	// No error path: Health degrades panel by panel and says which parts it
	// could not read. A console that answers 500 because Prometheus is down is
	// a console that is unavailable exactly when it is needed.
	httpx.JSON(w, http.StatusOK, s.Health(r.Context()))
}

func (s *Service) handleDeploy(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body struct {
		reasoned
		Ref string `json:"ref"`
	}
	if !decode(w, r, &body) {
		return
	}
	link, err := s.TriggerDeploy(r.Context(), sess, body.Ref, body.Reason)
	if err != nil {
		fail(w, err, "could not trigger the deployment")
		return
	}
	httpx.JSON(w, http.StatusAccepted, map[string]string{"status": "started", "url": link})
}

func (s *Service) handleRestoreTest(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body struct {
		reasoned
		Detail string `json:"detail"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := s.RecordRestoreTest(r.Context(), sess, body.Detail, body.Reason); err != nil {
		fail(w, err, "could not record the restore test")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

func (s *Service) handleUsage(w http.ResponseWriter, r *http.Request) {
	usage, err := s.UsageFor(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		fail(w, err, "could not read the usage")
		return
	}
	httpx.JSON(w, http.StatusOK, usage)
}

func (s *Service) handleUsageCSV(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "id")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="usage-`+tenantID+`.csv"`)
	if err := s.WriteUsageCSV(r.Context(), w, tenantID); err != nil {
		// The header is already on the wire by the time this can fail, so
		// there is no status left to send: the log is where this goes, and the
		// operator sees a short file.
		slog.Error("control plane: could not write the usage export", "error", err)
	}
}

func (s *Service) handleCatalogStatusRoute(w http.ResponseWriter, r *http.Request) {
	status := s.catalogStatus(r.Context())
	httpx.JSON(w, http.StatusOK, status)
}

func (s *Service) handleCatalogOverviewRoute(w http.ResponseWriter, r *http.Request) {
	status := s.catalogStatus(r.Context())
	httpx.JSON(w, http.StatusOK, map[string]any{
		"catalog":  status,
		"platform": s.version(r.Context()),
	})
}

func (s *Service) handleCatalogSyncRoute(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFrom(r.Context())
	var body reasoned
	if !decode(w, r, &body) {
		return
	}
	if s.syncCatalogFn == nil {
		httpx.Error(w, http.StatusNotImplemented, "this deployment reads its app catalog from a file; there is no registry to sync with")
		return
	}
	var changed bool
	err := s.Do(r.Context(), sess, Change{
		Action:     "catalog.sync",
		TargetType: "platform",
		TargetID:   "catalog",
		Reason:     body.Reason,
	}, func(ctx context.Context, tx pgx.Tx) error {
		var syncErr error
		changed, syncErr = s.syncCatalogFn(ctx)
		return syncErr
	})
	if err != nil {
		fail(w, err, "could not sync the catalog")
		return
	}
	status := "unchanged"
	if changed {
		status = "updated"
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": status, "changed": changed})
}
