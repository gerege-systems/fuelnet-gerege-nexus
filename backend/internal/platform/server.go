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
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/billing"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/contacts"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/developer_portal"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/documents"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/esign"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/gov_services"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/inventory"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/products"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/ai"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appinstaller"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/dan"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/eid"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/eidmongolia"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/emailverify"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/gerege"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/integration"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/mailer"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/observability"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/rbac"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/resilience"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/security"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/ssoprovider"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"
)

// PlatformVersion is the semver the app-store manifests are validated against.
const PlatformVersion = "1.0.0"

type Server struct {
	db             *pgxpool.Pool
	installer      *appinstaller.AppInstaller
	router         *chi.Mux
	sessions       *auth.SessionStore
	loginLimiter   *security.IPRateLimiter
	pollLimiter    *security.IPRateLimiter
	aiLimiter      *security.IPRateLimiter
	verifyLimiter  *security.IPRateLimiter
	asyncMailer    *mailer.AsyncOTPMailer
	emailVerify    *emailverify.Service
	copilotSvc     *ai.CopilotService
	forecaster     *ai.Forecaster
	eidSvc         *eid.EIDService
	danSvc         *dan.DANService
	ssoProvider    *ssoprovider.SSOProvider
	geregeSvc      *gerege.GeregeService
	integrationMgr *integration.Manager
	permissions    *rbac.SQLPermissionStore
	billingMod     *billing.BillingModule
	documentsMod   *documents.DocumentsModule
	govMod         *gov_services.Module
	devPortalMod   *developer_portal.DeveloperPortalModule
	contactsMod    *contacts.Module
	productsMod    *products.Module
	inventoryMod   *inventory.Module
	esignMod       *esign.Module
	eidMN          *eidmongolia.Service
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

	// Populate full manifests.
	//
	// A manifest that failed to load used to be replaced by a silent stub with
	// no dependencies, permissions or menus. Three shipped manifests were in
	// fact malformed (object instead of array for "dependencies", plain
	// strings instead of objects for "permissions") and nobody noticed: the
	// apps installed with an empty dependency graph and never contributed a
	// menu entry. Catalog integrity is now a startup error.
	catalogDir := filepath.Dir(catalogPath)
	catalog := make([]appcatalog.CatalogApp, 0, len(rawCatalog))
	for _, app := range rawCatalog {
		if !security.IsValidSlug(app.Slug) {
			return nil, fmt.Errorf("catalog app %q has an invalid slug %q", app.ID, app.Slug)
		}
		manifestPath := filepath.Join(catalogDir, "manifests", app.Slug+".json")
		manifest, err := appcatalog.LoadManifestFile(manifestPath, PlatformVersion)
		if err != nil {
			return nil, fmt.Errorf("load manifest for %s: %w", app.ID, err)
		}
		if manifest.ID != app.ID {
			return nil, fmt.Errorf("manifest %s declares id %q but the catalog entry is %q",
				manifestPath, manifest.ID, app.ID)
		}
		app.Manifest = manifest
		catalog = append(catalog, app)
	}

	installer := appinstaller.NewAppInstaller(db, catalog, PlatformVersion)

	// Keep the apps table in step with the catalog file. A missing row makes
	// installation fail on the app_installations foreign key, so this is a
	// startup concern, not a seeding concern. A cold database must not stop the
	// process from booting — /ready reports that separately.
	syncCtx, cancelSync := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelSync()
	if err := installer.SyncCatalog(syncCtx); err != nil {
		slog.Error("failed to sync app catalog into database", "error", err)
	}

	// Instantiate compile-time Go modules once. Each constructor registers the
	// module in the global app registry; calling them twice (here and again in
	// registerAppModuleRoutes) built two instances per app.
	// The integration manager is built before the modules that use it: esign
	// files finished documents through it and gov_services books meetings
	// through it, so it is a dependency of both rather than a peer.
	integrationMgr := integration.NewManager(db)

	contactsMod := contacts.New(db)
	productsMod := products.New(db)
	inventoryMod := inventory.New(db, false) // false = prevent negative stock
	billingMod := billing.New(db)
	documentsMod := documents.New(db)
	govMod := gov_services.New(db, integrationMgr)
	eidMN, err := eidmongolia.New(db)
	if err != nil {
		return nil, fmt.Errorf("eID Mongolia service: %w", err)
	}
	esignMod := esign.New(db, gerege.NewEsignService(), eidMN, integrationMgr)

	// Instantiate Async Mailer Queue
	syncMailer := mailer.NewSyncOTPMailer(os.Getenv("SMTP_HOST"), os.Getenv("SMTP_PORT"), os.Getenv("SMTP_FROM"), os.Getenv("SMTP_PASSWORD"))
	asyncMailer := mailer.NewAsyncOTPMailer(syncMailer, 2, 64, 3)

	ssoProvider := ssoprovider.NewSSOProvider(db)
	devPortalMod := developer_portal.NewDeveloperPortalModule(ssoProvider)

	s := &Server{
		db:           db,
		installer:    installer,
		router:       chi.NewRouter(),
		sessions:     auth.NewSessionStore(db, auth.DefaultSessionTTL),
		loginLimiter: newLoginLimiter(),
		pollLimiter:  newPollLimiter(),
		aiLimiter:    security.NewIPRateLimiter(rate.Limit(20.0/60.0), 10),
		// The per-caller allowance for /verify/send is metered per client in the
		// database. This is the cruder guard in front of it: an unauthenticated
		// flood should not reach the client lookup at all.
		// One per second sustained, twenty in a burst.
		verifyLimiter:  security.NewIPRateLimiter(rate.Limit(1), 20),
		asyncMailer:    asyncMailer,
		emailVerify:    emailverify.NewService(db, asyncMailer),
		copilotSvc:     ai.NewCopilotService(db),
		forecaster:     ai.NewForecaster(db),
		eidSvc:         eid.NewEIDService(),
		danSvc:         dan.NewDANService(),
		ssoProvider:    ssoProvider,
		geregeSvc:      gerege.NewGeregeService(),
		integrationMgr: integrationMgr,
		permissions:    rbac.NewSQLPermissionStore(db),
		billingMod:     billingMod,
		documentsMod:   documentsMod,
		govMod:         govMod,
		devPortalMod:   devPortalMod,
		contactsMod:    contactsMod,
		productsMod:    productsMod,
		inventoryMod:   inventoryMod,
		esignMod:       esignMod,
		eidMN:          eidMN,
	}

	s.setupRoutes()
	return s, nil
}

// StartBackgroundJobs launches the periodic work app modules need. It is
// separate from NewServer so a test can build a server without spawning
// goroutines, and it returns immediately — every job runs until ctx is
// cancelled at shutdown.
func (s *Server) StartBackgroundJobs(ctx context.Context) {
	s.esignMod.StartHousekeeping(ctx)
	s.eidMN.StartHousekeeping(ctx)
	// Abandoned connect attempts and the delivery log are the two integration
	// tables that only ever grow.
	s.integrationMgr.StartHousekeeping(ctx)
	// Links nobody followed have to stop being reported as outstanding, and the
	// verification trail is an audit record with a retention window, not a
	// mailing list.
	s.emailVerify.StartHousekeeping(ctx)
}

// EmailVerification is the platform's shared "prove this address" service.
//
// It is exposed rather than kept private because it belongs to every app
// module, not to the platform's own handlers: a module takes it in its
// constructor the way gov_services takes the integration manager, and calls
// Send with its own app id as the source. One flow, one audit trail, one place
// where token reuse and open redirects are gotten right.
func (s *Server) EmailVerification() *emailverify.Service { return s.emailVerify }

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
		AllowedHeaders:   []string{"Accept", "Accept-Language", "Authorization", "Content-Type", "X-Tenant-ID"},
		AllowCredentials: true,
	}))
	r.Use(security.CSRFMiddleware)

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

	// ORY Hydra Grade OpenID Connect & OAuth2 Provider Endpoints
	r.Get("/.well-known/openid-configuration", s.ssoProvider.HandleOIDCDiscovery)
	r.Get("/.well-known/jwks.json", s.ssoProvider.HandleJWKS)
	r.Post("/oauth2/token", s.ssoProvider.HandleTokenEndpoint)
	r.Post("/oauth2/introspect", s.ssoProvider.HandleIntrospectEndpoint)
	r.Post("/oauth2/revoke", s.ssoProvider.HandleRevokeEndpoint)

	// Platform API
	r.Route("/api/v1", func(api chi.Router) {
		// Auth with rate limiting
		api.With(security.RateLimitMiddleware(s.loginLimiter)).Post("/auth/login", s.handleLogin)
		api.With(security.RateLimitMiddleware(s.loginLimiter)).Post("/auth/eid/login", s.handleEIDLogin)
		api.With(security.RateLimitMiddleware(s.loginLimiter)).Post("/auth/eid/start", s.handleEIDStart)
		api.With(security.RateLimitMiddleware(s.loginLimiter)).Post("/auth/eid/start-id", s.handleEIDStartByNationalID)
		// Not the login limiter: a citizen polls for as long as it takes them to
		// reach their phone, and sharing that budget with sign-in attempts made
		// a busy office throttle itself out of signing in at all.
		api.With(security.RateLimitMiddleware(s.pollLimiter)).Post("/auth/eid/poll", s.handleEIDPoll)
		api.With(security.RateLimitMiddleware(s.loginLimiter)).Post("/auth/dan/login", s.handleDANLogin)
		api.Post("/auth/logout", s.handleLogout)

		// The OAuth redirect a connected provider sends the browser back to.
		// Unauthenticated on purpose — see handleIntegrationOAuthCallback: the
		// single-use state row is what carries the authority here, because a
		// cross-site redirect from Google cannot be relied on to still present
		// a SameSite=Strict session cookie.
		api.Get("/integrations/oauth/callback", s.handleIntegrationOAuthCallback)

		// Email verification. Both endpoints are outside the authenticated
		// group on purpose.
		//
		// /verify/send is the platform's shared "prove this address" service
		// offered to callers who have no session here — another platform, a
		// mobile backend, a partner — authenticated by a client key issued from
		// Settings. It still accepts a session, so the product's own screens do
		// not have to hold a key to call it.
		//
		// /verify/confirm is the link in the mail. The person following it is
		// the one being verified and has no account here; the single-use token
		// is the whole authority.
		api.With(security.RateLimitMiddleware(s.verifyLimiter)).Post("/verify/send", s.handleVerifySend)
		api.Get("/verify/confirm", s.handleVerifyConfirm)

		// Protected endpoints
		api.Group(func(pr chi.Router) {
			pr.Use(s.authMiddleware)

			pr.Get("/auth/me", s.handleMe)
			pr.Get("/menus", s.handleMenus)

			// Tenant access control. Mutations are deliberately admin-only;
			// authorization configuration can otherwise be used to self-elevate.
			pr.Route("/admin/access", func(ac chi.Router) {
				ac.Use(s.requireAdmin)
				ac.Get("/overview", s.handleAccessOverview)
				ac.Post("/roles", s.handleCreateRole)
				ac.Put("/roles/{id}", s.handleUpdateRole)
				ac.Delete("/roles/{id}", s.handleDeleteRole)
				ac.Put("/roles/{id}/permissions", s.handleSetRolePermissions)
				ac.Put("/memberships/{id}/roles", s.handleSetMembershipRoles)
			})

			// Email verification administration. Issuing a key that can send
			// mail in the tenant's name — and reading who has been written to —
			// is administrative, so it sits with the rest of the settings
			// surface rather than with the send endpoint it configures.
			pr.Route("/admin/email-verification", func(vr chi.Router) {
				vr.Use(s.requireAdmin)
				vr.Get("/overview", s.handleEmailVerifyOverview)
				vr.Get("/clients", s.handleListEmailVerifyClients)
				vr.Post("/clients", s.handleCreateEmailVerifyClient)
				vr.Put("/clients/{id}", s.handleUpdateEmailVerifyClient)
				vr.Delete("/clients/{id}", s.handleDeleteEmailVerifyClient)
			})

			// AI Copilot & Forecasting
			pr.With(security.RateLimitMiddleware(s.aiLimiter)).Post("/ai/copilot", s.handleAICopilot)
			pr.With(security.RateLimitMiddleware(s.aiLimiter)).Post("/ai/chat", s.handleAIChat)
			pr.With(security.RateLimitMiddleware(s.aiLimiter)).Post("/ai/stt", s.handleAISTT)
			pr.With(security.RateLimitMiddleware(s.aiLimiter)).Post("/ai/tts", s.handleAITTS)
			pr.With(security.RateLimitMiddleware(s.aiLimiter)).Post("/ai/translate", s.handleAITranslate)
			pr.Get("/ai/stock-forecast", s.handleAIForecast)
			pr.With(s.requireAdmin).Get("/admin/ai/prompts", s.handleAIListPrompts)
			pr.With(s.requireAdmin).Put("/admin/ai/prompts/{key}", s.handleAIUpdatePrompt)
			pr.With(s.requireAdmin).Get("/admin/ai/knowledge", s.handleAIListKnowledge)
			pr.With(s.requireAdmin).Post("/admin/ai/knowledge", s.handleAICreateKnowledge)

			// XYP State Information Exchange System (xyp.gerege.mn)
			// XYP responses contain authoritative citizen/company data. Merely
			// belonging to a tenant is not enough authority to query that data.
			pr.With(rbac.RequirePermission(s.permissions, "xyp.citizen.read")).Post("/xyp/citizen", s.handleXYPCitizenQuery)
			pr.With(rbac.RequirePermission(s.permissions, "xyp.company.read")).Post("/xyp/company", s.handleXYPCompanyQuery)

			// External Integrations Manager (admin-only: a connector target URL
			// makes the server issue arbitrary outbound requests, and an OAuth
			// grant hands the platform an account outside it)
			pr.Route("/integrations", func(ir chi.Router) {
				ir.Use(s.requireAdmin)
				ir.Get("/", s.handleListIntegrations)
				ir.Post("/", s.handleRegisterIntegration)
				ir.Get("/providers", s.handleIntegrationProviders)
				ir.Get("/deliveries", s.handleIntegrationDeliveries)
				ir.Put("/{id}", s.handleUpdateIntegration)
				ir.Delete("/{id}", s.handleDeleteIntegration)
				ir.Post("/{id}/connect", s.handleConnectIntegration)
				ir.Post("/{id}/disconnect", s.handleDisconnectIntegration)
			})

			// Store — reads are open to any tenant member, mutations are
			// tenant-administrator only. Previously every authenticated user
			// could install, enable or disable apps for the whole tenant.
			pr.Get("/store/apps", s.handleListStoreApps)
			pr.Get("/store/apps/{slug}", s.handleGetStoreApp)
			pr.Get("/installed-apps", s.handleListInstalledApps)

			pr.Group(func(ar chi.Router) {
				ar.Use(s.requireAdmin)
				ar.Post("/store/apps/{slug}/install", s.handleInstallApp)
				ar.Post("/store/apps/{slug}/enable", s.handleEnableApp)
				ar.Post("/store/apps/{slug}/disable", s.handleDisableApp)
			})
		})
	})

	// Register compile-time Business App Routes with Tenant & App Gate protection
	s.registerAppModuleRoutes()
}

// registerAppModuleRoutes mounts every compile-time business module behind the
// tenant app gate. Billing, Documents and the Developer Portal used to be wired
// straight into the protected group, so their endpoints stayed reachable for
// tenants that had never installed the app.
func (s *Server) registerAppModuleRoutes() {
	s.contactsMod.RegisterRoutes(s.router, s.appGateMiddleware("io.example.contacts"))
	s.productsMod.RegisterRoutes(s.router, s.appGateMiddleware("io.example.products"))
	s.inventoryMod.RegisterRoutes(s.router, s.appGateMiddleware("io.example.inventory"))
	s.billingMod.RegisterRoutes(s.router, s.appGateMiddleware("io.example.billing"))
	s.documentsMod.RegisterRoutes(s.router, s.appGateMiddleware("io.example.documents"))
	s.govMod.RegisterRoutes(s.router, s.appGateMiddleware("io.example.gov_services"))
	s.devPortalMod.RegisterRoutes(s.router, s.appGateMiddleware("io.example.developer_portal"))
	s.esignMod.RegisterRoutes(s.router, s.appGateMiddleware("io.example.esign"))
}

// Handlers
