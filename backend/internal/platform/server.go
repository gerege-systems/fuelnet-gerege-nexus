package platform

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/apps/contacts"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/apps/inventory"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/apps/products"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/ai"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/appcatalog"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/appinstaller"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/mailer"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/menu"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/observability"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/resilience"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/security"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/tenant"
	"golang.org/x/time/rate"
)

type Server struct {
	db           *pgxpool.Pool
	installer    *appinstaller.AppInstaller
	router       *chi.Mux
	loginLimiter *security.IPRateLimiter
	asyncMailer  *mailer.AsyncOTPMailer
	copilotSvc   *ai.CopilotService
	forecaster   *ai.Forecaster
}

func NewServer(db *pgxpool.Pool, catalogPath string) (*Server, error) {
	catalogData, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, err
	}

	var rawCatalog []appcatalog.CatalogApp
	if err := json.Unmarshal(catalogData, &rawCatalog); err != nil {
		return nil, err
	}

	// Populate full manifests
	catalog := make([]appcatalog.CatalogApp, 0, len(rawCatalog))
	for _, app := range rawCatalog {
		manifestPath := "catalog/manifests/" + app.Slug + ".json"
		manifest, err := appcatalog.LoadManifestFile(manifestPath, "1.0.0")
		if err != nil {
			manifest = appcatalog.Manifest{
				ID:       app.ID,
				Name:     app.Name,
				Version:  app.Version,
				Platform: ">=0.1.0",
			}
		}
		app.Manifest = manifest
		catalog = append(catalog, app)
	}

	installer := appinstaller.NewAppInstaller(db, catalog, "1.0.0")

	// Instantiate compile-time Go modules
	contacts.New(db)
	products.New(db)
	inventory.New(db, false) // false = prevent negative stock

	// Instantiate Async Mailer Queue
	syncMailer := mailer.NewSyncOTPMailer(os.Getenv("SMTP_HOST"), os.Getenv("SMTP_PORT"), os.Getenv("SMTP_FROM"), os.Getenv("SMTP_PASSWORD"))
	asyncMailer := mailer.NewAsyncOTPMailer(syncMailer, 2, 64, 3)

	s := &Server{
		db:           db,
		installer:    installer,
		router:       chi.NewRouter(),
		loginLimiter: security.NewIPRateLimiter(rate.Limit(5.0/60.0), 5), // 5 logins per minute
		asyncMailer:  asyncMailer,
		copilotSvc:   ai.NewCopilotService(db),
		forecaster:   ai.NewForecaster(db),
	}

	s.setupRoutes()
	return s, nil
}

func (s *Server) Router() *chi.Mux {
	return s.router
}

func (s *Server) setupRoutes() {
	r := s.router

	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(resilience.NewLoadShedder(1000).Middleware)
	r.Use(observability.MetricsMiddleware)
	r.Use(security.HeadersMiddleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   security.SafeCORSOrigins(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Tenant-ID"},
		AllowCredentials: true,
	}))

	// Infrastructure
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := s.db.Ping(r.Context()); err != nil {
			http.Error(w, `{"status":"error","message":"database unreachable"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	// Prometheus Metrics Endpoint
	r.Handle("/metrics", observability.MetricsHandler())

	// Platform API
	r.Route("/api/v1", func(api chi.Router) {
		// Auth with rate limiting
		api.With(security.RateLimitMiddleware(s.loginLimiter)).Post("/auth/login", s.handleLogin)
		api.Post("/auth/logout", s.handleLogout)

		// Protected endpoints
		api.Group(func(pr chi.Router) {
			pr.Use(s.authMiddleware)

			pr.Get("/auth/me", s.handleMe)
			pr.Get("/menus", s.handleMenus)

			// AI Copilot & Forecasting
			pr.Post("/ai/copilot", s.handleAICopilot)
			pr.Get("/ai/stock-forecast", s.handleAIForecast)

			// Store
			pr.Get("/store/apps", s.handleListStoreApps)
			pr.Get("/store/apps/{slug}", s.handleGetStoreApp)
			pr.Get("/installed-apps", s.handleListInstalledApps)
			pr.Post("/store/apps/{slug}/install", s.handleInstallApp)
			pr.Post("/store/apps/{slug}/enable", s.handleEnableApp)
			pr.Post("/store/apps/{slug}/disable", s.handleDisableApp)
		})
	})

	// Register compile-time Business App Routes with Tenant & App Gate protection
	s.registerAppModuleRoutes()
}

func (s *Server) registerAppModuleRoutes() {
	contactsMod, _ := contacts.New(s.db), true
	productsMod, _ := products.New(s.db), true
	inventoryMod, _ := inventory.New(s.db, false), true

	contactsMod.RegisterRoutes(s.router, s.appGateMiddleware("io.example.contacts"))
	productsMod.RegisterRoutes(s.router, s.appGateMiddleware("io.example.products"))
	inventoryMod.RegisterRoutes(s.router, s.appGateMiddleware("io.example.inventory"))
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cookie or Authorization header
		tokenStr := ""
		if cookie, err := r.Cookie("session_token"); err == nil {
			tokenStr = cookie.Value
		} else if header := r.Header.Get("Authorization"); len(header) > 7 && header[:7] == "Bearer " {
			tokenStr = header[7:]
		}

		if tokenStr == "" {
			http.Error(w, `{"error":"unauthorized: missing session token"}`, http.StatusUnauthorized)
			return
		}

		// Simple session token format: userId:tenantId:isAdmin
		var userID, tenantID string
		var isAdmin bool
		err := s.db.QueryRow(r.Context(),
			`SELECT u.id, m.tenant_id, u.is_admin 
			 FROM users u 
			 JOIN memberships m ON m.user_id = u.id 
			 WHERE u.id::text = $1 LIMIT 1`, tokenStr).Scan(&userID, &tenantID, &isAdmin)

		if err != nil {
			http.Error(w, `{"error":"unauthorized: invalid session"}`, http.StatusUnauthorized)
			return
		}

		ctx := auth.WithUserContext(r.Context(), auth.UserClaims{
			UserID:   userID,
			TenantID: tenantID,
			IsAdmin:  isAdmin,
		})
		ctx = tenant.WithTenantID(ctx, tenantID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) appGateMiddleware(appID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, err := tenant.FromContext(r.Context())
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Check if app is installed and enabled for this tenant
			var enabled bool
			err = s.db.QueryRow(r.Context(),
				`SELECT enabled FROM app_installations WHERE tenant_id = $1 AND app_id = $2`,
				tenantID, appID).Scan(&enabled)

			if err != nil || !enabled {
				http.Error(w, `{"error":"forbidden: app module `+appID+` is not installed or enabled for this tenant"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		}))
	}
}

// Handlers
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"invalid login credentials"}`, http.StatusBadRequest)
		return
	}

	var userID, passwordHash, tenantID, name string
	var isAdmin bool
	err := s.db.QueryRow(r.Context(),
		`SELECT u.id, u.password_hash, u.name, u.is_admin, m.tenant_id
		 FROM users u
		 JOIN memberships m ON m.user_id = u.id
		 WHERE u.email = $1 LIMIT 1`, req.Email).Scan(&userID, &passwordHash, &name, &isAdmin, &tenantID)

	if err != nil || !auth.CheckPasswordHash(req.Password, passwordHash) {
		audit.Record(r.Context(), "unknown", "anonymous", "auth.login_failed", "user", map[string]any{"email": req.Email})
		http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
		return
	}

	// Set secure HTTP-only session cookie
	isProd := os.Getenv("ENVIRONMENT") == "production"
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    userID,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteStrictMode,
	})

	audit.Record(r.Context(), tenantID, userID, "auth.login_success", "user", map[string]any{"email": req.Email})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token": userID,
		"user": map[string]any{
			"id":        userID,
			"tenant_id": tenantID,
			"name":      name,
			"email":     req.Email,
			"is_admin":  isAdmin,
		},
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	isProd := os.Getenv("ENVIRONMENT") == "production"
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteStrictMode,
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var name, email string
	var tenantName string
	_ = s.db.QueryRow(r.Context(), `SELECT name, email FROM users WHERE id = $1`, claims.UserID).Scan(&name, &email)
	_ = s.db.QueryRow(r.Context(), `SELECT name FROM tenants WHERE id = $1`, claims.TenantID).Scan(&tenantName)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":          claims.UserID,
		"tenant_id":   claims.TenantID,
		"tenant_name": tenantName,
		"name":        name,
		"email":       email,
		"is_admin":    claims.IsAdmin,
	})
}

func (s *Server) handleMenus(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	menus, err := menu.GetTenantMenus(r.Context(), s.installer, tenantID)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch menus"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(menus)
}

func (s *Server) handleListStoreApps(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := tenant.FromContext(r.Context())
	catalog := s.installer.GetCatalog()

	enabledIDs, _ := s.installer.GetEnabledAppIDsForTenant(r.Context(), tenantID)
	enabledMap := make(map[string]bool)
	for _, id := range enabledIDs {
		enabledMap[id] = true
	}

	type StoreAppResponse struct {
		appcatalog.CatalogApp
		Installed bool `json:"installed"`
		Enabled   bool `json:"enabled"`
	}

	res := make([]StoreAppResponse, 0, len(catalog))
	for _, app := range catalog {
		res = append(res, StoreAppResponse{
			CatalogApp: app,
			Installed:  enabledMap[app.ID],
			Enabled:    enabledMap[app.ID],
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handleGetStoreApp(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !security.IsValidSlug(slug) {
		http.Error(w, `{"error":"invalid app slug format"}`, http.StatusBadRequest)
		return
	}

	app, ok := s.installer.GetAppBySlug(slug)
	if !ok {
		http.Error(w, `{"error":"app not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(app)
}

func (s *Server) handleListInstalledApps(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	rows, err := s.db.Query(r.Context(),
		`SELECT ai.id, ai.app_id, a.slug, a.name, ai.installed_version, ai.status, ai.enabled, ai.installed_at
		 FROM app_installations ai
		 JOIN apps a ON a.id = ai.app_id
		 WHERE ai.tenant_id = $1`, tenantID)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type InstalledApp struct {
		ID               string    `json:"id"`
		AppID            string    `json:"app_id"`
		Slug             string    `json:"slug"`
		Name             string    `json:"name"`
		InstalledVersion string    `json:"installed_version"`
		Status           string    `json:"status"`
		Enabled          bool      `json:"enabled"`
		InstalledAt      time.Time `json:"installed_at"`
	}

	list := make([]InstalledApp, 0)
	for rows.Next() {
		var item InstalledApp
		if err := rows.Scan(&item.ID, &item.AppID, &item.Slug, &item.Name, &item.InstalledVersion, &item.Status, &item.Enabled, &item.InstalledAt); err == nil {
			list = append(list, item)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (s *Server) handleInstallApp(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !security.IsValidSlug(slug) {
		http.Error(w, `{"error":"invalid app slug format"}`, http.StatusBadRequest)
		return
	}

	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if err := s.installer.InstallApp(r.Context(), claims.TenantID, slug, claims.UserID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "installed", "app": slug})
}

func (s *Server) handleDisableApp(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !security.IsValidSlug(slug) {
		http.Error(w, `{"error":"invalid app slug format"}`, http.StatusBadRequest)
		return
	}

	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if err := s.installer.DisableApp(r.Context(), claims.TenantID, slug, claims.UserID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "disabled", "app": slug})
}

func (s *Server) handleEnableApp(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if !security.IsValidSlug(slug) {
		http.Error(w, `{"error":"invalid app slug format"}`, http.StatusBadRequest)
		return
	}

	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if err := s.installer.EnableApp(r.Context(), claims.TenantID, slug, claims.UserID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "enabled", "app": slug})
}

func (s *Server) handleAICopilot(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Prompt == "" {
		http.Error(w, `{"error":"invalid prompt"}`, http.StatusBadRequest)
		return
	}

	res, err := s.copilotSvc.Query(r.Context(), ai.CopilotRequest{
		Prompt:   req.Prompt,
		TenantID: tenantID,
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handleAIForecast(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	forecast, err := s.forecaster.AnalyzeTenantStock(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error":"failed to generate forecast"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(forecast)
}
