/*
 * Gerege Nexus — App Store registry
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

package appstore

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/security"
	"github.com/go-chi/chi/v5"
)

type callerKey struct{}

// requireCaller resolves the Gerege identity behind a request.
func (s *Server) requireCaller(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
			httpx.Error(w, http.StatusUnauthorized, "sign in with Gerege to use the developer API")
			return
		}

		caller, err := s.verifier.Verify(r.Context(), strings.TrimSpace(header[len(prefix):]))
		if err != nil {
			// The reason a token was refused is the caller's business only in
			// so far as "sign in again" — the detail belongs in the log, where
			// an operator debugging a console can see it.
			slog.Info("rejected a developer token", "error", err)
			httpx.Error(w, http.StatusUnauthorized, "your session is not valid; sign in again")
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), callerKey{}, caller)))
	})
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if caller := callerFrom(r.Context()); caller == nil || !caller.Admin {
			httpx.Error(w, http.StatusForbidden, "this is a registry administrator action")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func callerFrom(ctx context.Context) *Caller {
	caller, _ := ctx.Value(callerKey{}).(*Caller)
	return caller
}

// --- developer --------------------------------------------------------------

// handleDevMe describes the signed-in developer and the publisher they act for.
// A console's first call: it decides whether to show the registration form or
// the app list.
func (s *Server) handleDevMe(w http.ResponseWriter, r *http.Request) {
	caller := callerFrom(r.Context())
	response := map[string]any{
		"subject": caller.Subject, "email": caller.Email, "name": caller.Name,
		"tenant_slug": caller.TenantSlug, "admin": caller.Admin,
	}

	publisher, err := s.store.PublisherByOwner(r.Context(), caller.Subject)
	switch {
	case errors.Is(err, ErrNotFound):
		response["publisher"] = nil
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "could not load your publisher")
		return
	default:
		response["publisher"] = publisher
	}
	httpx.JSON(w, http.StatusOK, response)
}

func (s *Server) handleCreatePublisher(w http.ResponseWriter, r *http.Request) {
	caller := callerFrom(r.Context())

	var body struct {
		Slug         string `json:"slug"`
		Name         string `json:"name"`
		ContactEmail string `json:"contact_email"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "malformed request body")
		return
	}
	if !security.IsValidSlug(body.Slug) || body.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "a publisher needs a name and a slug of lowercase letters, digits, - and _")
		return
	}

	// One publisher per person, for now. A team belongs to an organisation
	// rather than to whoever registered first, and modelling that properly is
	// worth more than allowing a second row here.
	if _, err := s.store.PublisherByOwner(r.Context(), caller.Subject); err == nil {
		httpx.Error(w, http.StatusConflict, "you already have a publisher account")
		return
	}

	email := body.ContactEmail
	if email == "" {
		email = caller.Email
	}
	publisher, err := s.store.CreatePublisher(r.Context(), &Publisher{
		Slug: body.Slug, Name: body.Name, ContactEmail: email,
		OwnerSub: caller.Subject, OwnerEmail: caller.Email,
		OwnerTenantID: caller.TenantID, OwnerTenantSlug: caller.TenantSlug,
	})
	if errors.Is(err, ErrConflict) {
		httpx.Error(w, http.StatusConflict, "that publisher slug is taken")
		return
	}
	if err != nil {
		slog.Error("could not create a publisher", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "could not create the publisher")
		return
	}
	httpx.JSON(w, http.StatusCreated, publisher)
}

// publisherFor resolves the caller's publisher or answers for them.
func (s *Server) publisherFor(w http.ResponseWriter, r *http.Request) *Publisher {
	publisher, err := s.store.PublisherByOwner(r.Context(), callerFrom(r.Context()).Subject)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusForbidden, "register a publisher account first")
		return nil
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load your publisher")
		return nil
	}
	return publisher
}

func (s *Server) handleDevApps(w http.ResponseWriter, r *http.Request) {
	publisher := s.publisherFor(w, r)
	if publisher == nil {
		return
	}
	apps, err := s.store.ListApps(r.Context(), publisher.ID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not list your apps")
		return
	}
	httpx.JSON(w, http.StatusOK, apps)
}

// handleDevUpsertApp registers a catalogue entry, or edits one the caller owns.
func (s *Server) handleDevUpsertApp(w http.ResponseWriter, r *http.Request) {
	publisher := s.publisherFor(w, r)
	if publisher == nil {
		return
	}

	var body struct {
		ID           string                               `json:"id"`
		Slug         string                               `json:"slug"`
		Type         string                               `json:"type"`
		Name         string                               `json:"name"`
		Description  string                               `json:"description"`
		IconURL      string                               `json:"icon_url"`
		Category     string                               `json:"category"`
		Visibility   string                               `json:"visibility"`
		Translations map[string]appcatalog.CatalogAppText `json:"translations"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<18)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "malformed request body")
		return
	}
	if body.ID == "" || body.Name == "" || !security.IsValidSlug(body.Slug) {
		httpx.Error(w, http.StatusBadRequest, "an app needs an id, a name and a valid slug")
		return
	}
	if body.Type == "" {
		body.Type = appcatalog.TypeModule
	}
	if body.Type != appcatalog.TypeModule && body.Type != appcatalog.TypeExternal {
		httpx.Error(w, http.StatusBadRequest, `type must be "module" or "external"`)
		return
	}
	if body.Visibility == "" {
		body.Visibility = "public"
	}
	if body.Category == "" {
		body.Category = "General"
	}

	app := &App{
		ID: body.ID, PublisherID: publisher.ID, Slug: body.Slug, Type: body.Type,
		Name: body.Name, Description: body.Description, IconURL: body.IconURL,
		Category: body.Category, Visibility: body.Visibility,
	}
	if err := s.store.UpsertApp(r.Context(), app, body.Translations); errors.Is(err, ErrConflict) {
		httpx.Error(w, http.StatusConflict, "that app id or slug belongs to another publisher")
		return
	} else if err != nil {
		slog.Error("could not save an app", "error", err, "app_id", body.ID)
		httpx.Error(w, http.StatusInternalServerError, "could not save the app")
		return
	}

	saved, err := s.store.AppByID(r.Context(), body.ID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not reload the app")
		return
	}
	httpx.JSON(w, http.StatusOK, saved)
}

// devApp resolves an app the caller is allowed to act on.
func (s *Server) devApp(w http.ResponseWriter, r *http.Request) (*App, bool) {
	publisher := s.publisherFor(w, r)
	if publisher == nil {
		return nil, false
	}
	app, err := s.store.AppBySlug(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "app not found")
		return nil, false
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load the app")
		return nil, false
	}
	// 404 rather than 403: whether another publisher's app exists is not this
	// caller's business.
	if app.PublisherID != publisher.ID {
		httpx.Error(w, http.StatusNotFound, "app not found")
		return nil, false
	}
	return app, true
}

func (s *Server) handleDevVersions(w http.ResponseWriter, r *http.Request) {
	app, ok := s.devApp(w, r)
	if !ok {
		return
	}
	versions, err := s.store.ListVersions(r.Context(), app.ID, false)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load versions")
		return
	}
	httpx.JSON(w, http.StatusOK, versions)
}

// handleDevSubmitVersion accepts a manifest for review.
//
// The manifest is validated here, against the same rules a Nexus instance
// applies when it receives the catalogue. Publishing something an instance
// would reject does not break the instance — it discards the whole document and
// keeps what it has — but it stops every instance receiving updates until
// somebody notices, so the check belongs at the point of submission where a
// person is present to read the error.
func (s *Server) handleDevSubmitVersion(w http.ResponseWriter, r *http.Request) {
	app, ok := s.devApp(w, r)
	if !ok {
		return
	}

	var body struct {
		Channel  string              `json:"channel"`
		Manifest appcatalog.Manifest `json:"manifest"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "malformed request body")
		return
	}
	if body.Channel == "" {
		body.Channel = "stable"
	}
	if body.Channel != "stable" && body.Channel != "beta" {
		httpx.Error(w, http.StatusBadRequest, `channel must be "stable" or "beta"`)
		return
	}
	if body.Manifest.ID != app.ID {
		httpx.Error(w, http.StatusBadRequest, "the manifest id must match the app id")
		return
	}
	if (body.Manifest.Type == appcatalog.TypeExternal) != (app.Type == appcatalog.TypeExternal) {
		httpx.Error(w, http.StatusBadRequest, "the manifest type must match the app type")
		return
	}
	// Validated with an empty platform version, so the app's own platform
	// constraint is checked for sanity without being checked against this
	// service's version — the registry is not a Nexus instance and has no
	// platform version of its own to compare against.
	if err := appcatalog.ValidateManifest(body.Manifest, ""); err != nil {
		httpx.Error(w, http.StatusBadRequest, "manifest rejected: "+err.Error())
		return
	}

	version := &Version{
		AppID: app.ID, Version: body.Manifest.Version, Channel: body.Channel,
		MinPlatform: body.Manifest.Platform, Manifest: body.Manifest,
		Status: StatusInReview, SubmittedBy: callerFrom(r.Context()).Email,
	}
	if version.MinPlatform == "" {
		version.MinPlatform = ">=0.1.0"
	}

	var external *ExternalRegistration
	if body.Manifest.IsExternal() && body.Manifest.External != nil {
		external = &ExternalRegistration{
			AppID:       app.ID,
			LaunchURL:   body.Manifest.External.LaunchURL,
			SSOClientID: body.Manifest.External.SSOClientID,
			Scopes:      body.Manifest.External.Scopes,
			Embed:       body.Manifest.External.Embed,
			HealthURL:   body.Manifest.External.HealthURL,
		}
		if external.Embed == "" {
			external.Embed = "new_tab"
		}
	}

	saved, err := s.store.SubmitVersion(r.Context(), version, external)
	if errors.Is(err, ErrConflict) {
		httpx.Error(w, http.StatusConflict,
			"that version already exists; a published version is immutable, so publish a new number")
		return
	}
	if err != nil {
		slog.Error("could not submit a version", "error", err, "app_id", app.ID)
		httpx.Error(w, http.StatusInternalServerError, "could not submit the version")
		return
	}
	httpx.JSON(w, http.StatusCreated, saved)
}

// --- administration ---------------------------------------------------------

func (s *Server) handleReviewQueue(w http.ResponseWriter, r *http.Request) {
	queue, err := s.store.ListReviewQueue(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load the review queue")
		return
	}
	httpx.JSON(w, http.StatusOK, queue)
}

// handleReviewDecision publishes, rejects or yanks one submitted version.
func (s *Server) handleReviewDecision(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Action string `json:"action"`
		Note   string `json:"note"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "malformed request body")
		return
	}

	caller := callerFrom(r.Context())
	actor := caller.Email
	if actor == "" {
		actor = caller.Subject
	}

	err := s.store.DecideVersion(r.Context(), chi.URLParam(r, "id"), body.Action, actor, body.Note)
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "version not found")
		return
	case err != nil && strings.HasPrefix(err.Error(), "unknown review action"):
		httpx.Error(w, http.StatusBadRequest, "action must be publish, reject or yank")
		return
	case err != nil:
		slog.Error("review decision failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "could not record the decision")
		return
	}

	slog.Info("catalog changed", "action", body.Action, "version_id", chi.URLParam(r, "id"), "actor", actor)
	httpx.JSON(w, http.StatusOK, map[string]string{"status": body.Action})
}

func (s *Server) handleAdminPublishers(w http.ResponseWriter, r *http.Request) {
	publishers, err := s.store.ListPublishers(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not list publishers")
		return
	}
	httpx.JSON(w, http.StatusOK, publishers)
}

func (s *Server) handleVerifyPublisher(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Verified bool `json:"verified"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "malformed request body")
		return
	}
	if err := s.store.SetPublisherVerified(r.Context(), chi.URLParam(r, "id"), body.Verified); errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "publisher not found")
		return
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not update the publisher")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"verified": body.Verified})
}
