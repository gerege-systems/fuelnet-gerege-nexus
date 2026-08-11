/*
 * Gerege Nexus — App Store registry
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

package appstore

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/security"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server is the registry's HTTP surface.
//
// Three audiences, three shapes of authority:
//
//	/api/v1/registry/…   every Nexus instance and the storefront. No sign-in at
//	                     all — a public catalogue that needed an account would
//	                     not be a storefront, and the signature is what makes
//	                     the contents trustworthy rather than the transport.
//	/api/v1/dev/…        a publisher, identified by a Gerege id_token.
//	/api/v1/admin/…      whoever the deployment names as a reviewer.
type Server struct {
	db       *pgxpool.Pool
	store    *Store
	catalog  *CatalogService
	verifier *Verifier
	router   *chi.Mux
	origin   string
}

type Config struct {
	Origin            string
	StorefrontOrigins []string
	Issuer            string
	ConsoleAudience   string
	AdminSubjects     []string
	AdminEmails       []string
}

func NewServer(db *pgxpool.Pool, signer *Signer, cfg Config) *Server {
	store := NewStore(db)
	s := &Server{
		db:       db,
		store:    store,
		catalog:  NewCatalogService(store, signer),
		verifier: NewVerifier(cfg.Issuer, cfg.ConsoleAudience, cfg.AdminSubjects, cfg.AdminEmails),
		router:   chi.NewRouter(),
		origin:   cfg.Origin,
	}
	s.routes(cfg)
	return s
}

func (s *Server) Router() *chi.Mux { return s.router }

func (s *Server) routes(cfg Config) {
	r := s.router
	r.Use(chimiddleware.Recoverer)
	r.Use(security.HeadersMiddleware)

	// The registry read path is meant to be called from other origins — a
	// storefront, a console, and eventually anything that wants to show what is
	// published. It carries no cookie and no credential, so a permissive
	// allow-list here grants an attacker exactly what curl already grants them.
	allowed := append([]string{}, cfg.StorefrontOrigins...)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowed,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Accept-Language", "Authorization", "Content-Type", "If-None-Match"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "appstore-registry"})
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := s.db.Ping(r.Context()); err != nil {
			httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error"})
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	// The public keys a catalogue may be signed with. Instances pin one key by
	// value; this is what lets them follow a rotation without a redeploy.
	r.Get("/.well-known/appstore-keys.json", s.handleKeys)

	r.Route("/api/v1/registry", func(pub chi.Router) {
		pub.Get("/catalog", s.handleCatalog)
		pub.Get("/apps", s.handleListApps)
		pub.Get("/apps/{slug}", s.handleGetApp)
		pub.Get("/apps/{slug}/versions", s.handleListVersions)
	})

	r.Route("/api/v1/dev", func(dev chi.Router) {
		dev.Use(s.requireCaller)
		dev.Get("/me", s.handleDevMe)
		dev.Post("/publishers", s.handleCreatePublisher)
		dev.Get("/apps", s.handleDevApps)
		dev.Post("/apps", s.handleDevUpsertApp)
		dev.Get("/apps/{slug}/versions", s.handleDevVersions)
		dev.Post("/apps/{slug}/versions", s.handleDevSubmitVersion)
	})

	r.Route("/api/v1/admin", func(admin chi.Router) {
		admin.Use(s.requireCaller, s.requireAdmin)
		admin.Get("/review", s.handleReviewQueue)
		admin.Post("/review/{id}", s.handleReviewDecision)
		admin.Get("/publishers", s.handleAdminPublishers)
		admin.Post("/publishers/{id}/verify", s.handleVerifyPublisher)
	})
}

// --- public -----------------------------------------------------------------

func (s *Server) handleKeys(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.catalog.Catalog(r.Context(), "stable", "")
	_ = snapshot
	if err != nil {
		slog.Warn("keys endpoint could not warm the catalog", "error", err)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"keys": []map[string]string{{
			"key_id": s.catalog.signer.KeyID,
			"alg":    "Ed25519",
			"public": s.catalog.signer.PublicKey(),
		}},
	})
}

// handleCatalog is the endpoint every Nexus instance polls.
//
// It answers with bytes that were signed as a unit and stored that way; nothing
// here re-encodes them. The ETag is a hash of exactly those bytes, so a 304 is
// a statement about the document rather than about this request.
func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	platform := r.URL.Query().Get("platform")

	// A platform that is not a version is the caller's mistake. It used to
	// come back as a 500, which puts somebody's typo in the same alert as a
	// database that has gone away.
	if platform != "" {
		if _, err := semver.NewVersion(platform); err != nil {
			httpx.Error(w, http.StatusBadRequest, "platform must be a semver version, for example 1.0.0")
			return
		}
	}
	if channel != "" && channel != "stable" && channel != "beta" {
		httpx.Error(w, http.StatusBadRequest, `channel must be "stable" or "beta"`)
		return
	}

	snapshot, err := s.catalog.Catalog(r.Context(), channel, platform)
	if err != nil {
		slog.Error("could not build the catalog", "error", err, "channel", channel, "platform", platform)
		httpx.Error(w, http.StatusInternalServerError, "could not build the catalog")
		return
	}

	w.Header().Set("ETag", snapshot.ETag)
	w.Header().Set("Cache-Control", "public, max-age=60")
	if match := r.Header.Get("If-None-Match"); match != "" && etagMatches(match, snapshot.ETag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(snapshot.Document); err != nil {
		slog.Warn("catalog response was cut short", "error", err)
	}
}

// etagMatches implements the part of RFC 9110 §13.1.2 that matters here: a
// client may send several, and a proxy may have made ours weak.
func etagMatches(header, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag || strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}

func (s *Server) handleListApps(w http.ResponseWriter, r *http.Request) {
	apps, err := s.store.ListApps(r.Context(), "")
	if err != nil {
		slog.Error("could not list apps", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "could not list apps")
		return
	}
	locale := localeFrom(r)
	view := make([]map[string]any, 0, len(apps))
	for _, app := range apps {
		view = append(view, s.appView(r.Context(), app, locale))
	}
	httpx.JSON(w, http.StatusOK, view)
}

func (s *Server) handleGetApp(w http.ResponseWriter, r *http.Request) {
	app, err := s.store.AppBySlug(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "app not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load the app")
		return
	}

	versions, err := s.store.ListVersions(r.Context(), app.ID, true)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load versions")
		return
	}

	view := s.appView(r.Context(), *app, localeFrom(r))
	view["versions"] = versions
	if len(versions) > 0 {
		view["manifest"] = versions[0].Manifest
	}
	if external, err := s.store.ExternalRegistration(r.Context(), app.ID); err == nil {
		view["external"] = external
	}
	httpx.JSON(w, http.StatusOK, view)
}

func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	app, err := s.store.AppBySlug(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "app not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load the app")
		return
	}
	versions, err := s.store.ListVersions(r.Context(), app.ID, true)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load versions")
		return
	}
	httpx.JSON(w, http.StatusOK, versions)
}

// appView renders a catalogue entry in the caller's language.
func (s *Server) appView(ctx context.Context, app App, locale string) map[string]any {
	view := map[string]any{
		"id": app.ID, "slug": app.Slug, "type": app.Type, "name": app.Name,
		"description": app.Description, "icon_url": app.IconURL, "category": app.Category,
		"publisher": app.PublisherName, "latest_version": app.LatestVersion,
		"updated_at": app.UpdatedAt,
	}
	texts, err := s.store.AppTexts(ctx, app.ID)
	if err != nil {
		return view
	}
	view["translations"] = texts
	if text, ok := texts[locale]; ok {
		if text.Name != "" {
			view["name"] = text.Name
		}
		if text.Description != "" {
			view["description"] = text.Description
		}
		if text.Category != "" {
			view["category"] = text.Category
		}
	}
	return view
}

// localeFrom reads the language the caller asked for. The storefront passes it
// explicitly; a browser hitting the API directly gets Accept-Language.
func localeFrom(r *http.Request) string {
	if locale := r.URL.Query().Get("locale"); locale != "" {
		return locale
	}
	header := r.Header.Get("Accept-Language")
	if header == "" {
		return "en"
	}
	first := strings.SplitN(header, ",", 2)[0]
	return strings.ToLower(strings.TrimSpace(strings.SplitN(first, "-", 2)[0]))
}
