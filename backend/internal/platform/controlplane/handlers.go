package controlplane

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/security"
	"github.com/go-chi/chi/v5"
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
