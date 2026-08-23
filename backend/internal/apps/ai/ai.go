/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// Package ai is the assistant: the copilot, speech, translation, and the
// prompts and knowledge an organisation writes for it.
//
// It was the platform's until 2026-08-23 — ten routes in server.go, handlers in
// ai_handlers.go, the service under internal/platform/ai — which meant every
// deployment served an assistant whether it wanted one or not, and no
// deployment could remove it. That is the definition of an app in this
// architecture (docs/CORE_BOUNDARY_PLAN.md §4.2), so it is one now: installable
// from the store, removable from it, and gated per organisation like the rest.
//
// What stayed behind is the rail underneath, and the split is the useful part:
// the model key, the request budget and the monthly allowance are the
// deployment's, and the platform keeps them — the limiter through
// nexus.RateLimit, the allowance through nexus.QuotaGate. A module that
// enforced its own would be enforcing a number nobody sold.
package ai

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// ID is the app id, the same one the catalogue and the store carry.
const ID = "io.gerege.nexus.ai"

// The permissions this app declares.
//
// Two, and the line between them is who is affected. Asking the assistant
// something spends the organisation's allowance and reads what it is allowed to
// read; writing the prompt decides what the assistant is for every member of
// the organisation and what it is told about them.
const (
	PermissionUse    = "ai.read"
	PermissionManage = "ai.manage"
)

// The request budget for one caller, in requests per minute.
//
// A model call is expensive in a way an ordinary read is not — money per call,
// seconds per call — so the assistant carries a limit of its own rather than
// the platform's default. The numbers are the ones server.go enforced before
// the app left it, unchanged: twenty a minute is more than a person asking
// questions notices and less than a script wants.
const (
	requestsPerMinute = 20
	requestBurst      = 10
)

// Module is the app.
//
// The database handle is a field because two screens are stored settings rather
// than model traffic — the prompts an organisation overrides and the knowledge
// it pastes in — and both are ordinary rows in this deployment's database.
type Module struct {
	db      nexus.DB
	perms   nexus.PermissionStore
	copilot *CopilotService
}

// New builds the module and registers it.
func New(p nexus.Platform) *Module {
	m := &Module{
		db:      p.DB(),
		perms:   p.Permissions(),
		copilot: NewCopilotService(p.DB()),
	}
	nexus.Register(m)
	return m
}

func (m *Module) ID() string      { return ID }
func (m *Module) Name() string    { return "AI Assistant" }
func (m *Module) Version() string { return "1.0.0" }

// Dependencies is empty. The assistant answers from whatever tools the
// installed apps lend it (nexus.ProvideAssistant) and says so honestly when
// there are none — depending on one app would make it uninstallable without it
// for no gain.
func (m *Module) Dependencies() []nexus.Dependency { return nil }

func (m *Module) Permissions() []nexus.PermissionDefinition {
	return []nexus.PermissionDefinition{
		{Code: PermissionUse, Name: "Use the AI Assistant",
			Description:  "Ask the assistant, dictate, have text read out and translated",
			DefaultRoles: []string{nexus.DefaultRoleManager, nexus.DefaultRoleUser}},
		{Code: PermissionManage, Name: "Manage the AI Assistant",
			Description:  "Write the prompt the assistant follows and the knowledge it is given about this organisation",
			DefaultRoles: []string{nexus.DefaultRoleManager}},
	}
}

// Menus is empty, and that is what the shell already does: the assistant is
// reached from the chat affordance rather than from the sidebar, and its two
// settings screens do not exist in this repository's shell at all — the API
// client is there, nothing imports it. A menu entry pointing at a page nobody
// has written is a dead link in every deployment.
func (m *Module) Menus() []nexus.MenuDefinition { return nil }

// RegisterRoutes mounts the API behind the app gate.
//
// The URLs are unchanged — /api/v1/ai/* — because the shell's client already
// speaks them and a rename would be a second change wearing this one's clothes.
// What changed is what stands in front of them: the app gate, so an
// organisation without the app installed gets 404 rather than an assistant it
// never asked for, and a permission per route.
//
// The order of the middleware is the order the costs arrive in: the gate
// (is this app even here), the permission (may this person), the rate limit
// (how fast), the quota (how much this month). Nothing reaches the model until
// all four agree.
func (m *Module) RegisterRoutes(r chi.Router, gateMiddleware func(http.Handler) http.Handler) {
	limit := nexus.RateLimit(requestsPerMinute, requestBurst)
	quota := nexus.QuotaGate("ai")

	r.Route("/api/v1/ai", func(rr chi.Router) {
		rr.Use(gateMiddleware)

		// The model traffic. Every one of these spends the organisation's
		// monthly allowance, which is why the quota gate is on all of them and
		// not only on the expensive-looking ones.
		ask := rr.With(nexus.RequirePermission(m.perms, PermissionUse), limit, quota)
		ask.Post("/copilot", m.handleAICopilot)
		ask.Post("/chat", m.handleAIChat)
		ask.Post("/stt", m.handleAISTT)
		ask.Post("/tts", m.handleAITTS)
		ask.Post("/translate", m.handleAITranslate)

		// No model call of its own: it asks whichever app lends a stock
		// forecast for one. Still metered, because on a deployment where that
		// app computes it with a model the cost is the same one.
		rr.With(nexus.RequirePermission(m.perms, PermissionUse), quota).
			Get("/stock-forecast", m.handleAIForecast)
	})

	// The settings, under their own prefix rather than /api/v1/ai/admin: these
	// are administrative screens of one app, and /admin is where this platform
	// puts them for every app.
	r.Route("/api/v1/admin/ai", func(rr chi.Router) {
		rr.Use(gateMiddleware)
		rr.Use(nexus.RequirePermission(m.perms, PermissionManage))

		rr.Get("/prompts", m.handleAIListPrompts)
		rr.Put("/prompts/{key}", m.handleAIUpdatePrompt)
		rr.Get("/knowledge", m.handleAIListKnowledge)
		rr.Post("/knowledge", m.handleAICreateKnowledge)
	})
}
