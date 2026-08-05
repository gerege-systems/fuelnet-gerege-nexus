# Module Authoring Guide

Welcome to the **open-gerege-mn-erp** Module Authoring Guide! This guide explains how external developers can write, register, and distribute custom business application modules for the platform.

---

## 🏗️ Module Architecture Overview

In `open-gerege-mn-erp`, business modules are written in Go as compile-time packages under `backend/internal/apps/`. 

Every module MUST implement the `Module` interface defined in [`backend/internal/module.go`](../backend/internal/module.go):

```go
type Module interface {
    ID() string
    Name() string
    Version() string
    Dependencies() []Dependency
    Permissions() []PermissionDefinition
    Menus() []MenuDefinition
    RegisterRoutes(r chi.Router, gateMiddleware func(http.Handler) http.Handler)
}
```

---

## 📝 Step-by-Step: Creating a New Module

### Step 1: Define Module Struct & Register in `appregistry`
Create a new directory `backend/internal/apps/invoices/invoices.go`:

```go
package invoices

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/gerege-systems/open-gerege-mn-erp/backend/internal"
    "github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/appregistry"
)

type Module struct {
    db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Module {
    m := &Module{db: db}
    appregistry.Register(m)
    return m
}

func (m *Module) ID() string      { return "io.example.invoices" }
func (m *Module) Name() string    { return "Invoicing & Billing" }
func (m *Module) Version() string { return "1.0.0" }

func (m *Module) Dependencies() []internal.Dependency {
    return []internal.Dependency{
        {AppID: "io.example.contacts", VersionConstraint: ">=1.0.0"},
        {AppID: "io.example.products", VersionConstraint: ">=1.0.0"},
    }
}
```

### Step 2: Define Permissions and Menus
```go
func (m *Module) Permissions() []internal.PermissionDefinition {
    return []internal.PermissionDefinition{
        {Code: "invoices.read", Name: "View Invoices"},
        {Code: "invoices.manage", Name: "Create & Edit Invoices"},
    }
}

func (m *Module) Menus() []internal.MenuDefinition {
    return []internal.MenuDefinition{
        {ID: "menu_invoices", Label: "Invoices", Path: "/invoices", Icon: "file-text", Order: 30},
    }
}
```

### Step 3: Register HTTP Routes with App Gate Middleware
```go
func (m *Module) RegisterRoutes(r chi.Router, gateMiddleware func(http.Handler) http.Handler) {
    r.Route("/api/v1/invoices", func(sub chi.Router) {
        sub.Use(gateMiddleware)
        sub.Get("/", m.handleListInvoices)
        sub.Post("/", m.handleCreateInvoice)
    })
}
```

### Step 4: Create App Manifest JSON
Add a manifest file in `catalog/manifests/invoices.json`:

```json
{
  "id": "io.example.invoices",
  "name": "Invoices",
  "version": "1.0.0",
  "description": "Invoicing, customer billing, and payment tracking.",
  "category": "Accounting",
  "dependencies": [
    {"app_id": "io.example.contacts", "version_constraint": ">=1.0.0"},
    {"app_id": "io.example.products", "version_constraint": ">=1.0.0"}
  ]
}
```

And update `catalog/apps.json` to index the new app in the App Store!

---

## 👥 Authors
- **Gerege Systems Development Team & Gemini AI**
