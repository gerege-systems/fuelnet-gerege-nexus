package products

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Product struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	SKU       string    `json:"sku"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Module struct {
	db nexus.DB
}

func New(p nexus.Platform) *Module {
	m := &Module{db: p.DB()}
	nexus.Register(m)
	return m
}

func (m *Module) ID() string { return "io.gerege.nexus.products" }

// MenuPermission and RoutePermissionPrefix are this module's half of
// nexus.AccessPolicy — what the platform used to hold in a switch keyed by
// app ID, stated here so it survives the module moving to another repository.
func (m *Module) MenuPermission() string        { return "products.read" }
func (m *Module) RoutePermissionPrefix() string { return "products" }
func (m *Module) Name() string                  { return "Products" }
func (m *Module) Version() string               { return "1.0.0" }

func (m *Module) Dependencies() []nexus.Dependency {
	return nil
}

func (m *Module) Permissions() []nexus.PermissionDefinition {
	return []nexus.PermissionDefinition{
		{Code: "products.read", Name: "Read Products", Description: "View product catalog"},
		{Code: "products.manage", Name: "Manage Products", Description: "Create and edit products"},
	}
}

func (m *Module) Menus() []nexus.MenuDefinition {
	return []nexus.MenuDefinition{
		{ID: "products", ParentID: "master_data", Label: "Products", Path: "/products", Icon: "package", Order: 20, Labels: map[string]string{"mn": "Бараа бүтээгдэхүүн", "ar": "المنتجات", "zh": "产品", "fr": "Produits", "ru": "Товары", "es": "Productos"}},
	}
}

func (m *Module) RegisterRoutes(r chi.Router, tenantAuthMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/products", func(pr chi.Router) {
		pr.Use(tenantAuthMiddleware)
		pr.Get("/", m.listProductsHandler)
		pr.Post("/", m.createProductHandler)
		pr.Put("/{id}", m.updateProductHandler)
	})
}

func (m *Module) listProductsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	rows, err := m.db.Query(r.Context(),
		`SELECT id, tenant_id, sku, name, price, active, created_at, updated_at 
		 FROM products WHERE tenant_id = $1 ORDER BY name ASC`, tenantID)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := make([]Product, 0)
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.TenantID, &p.SKU, &p.Name, &p.Price, &p.Active, &p.CreatedAt, &p.UpdatedAt); err != nil {
			nexus.Error(w, http.StatusInternalServerError, "scan error")
			return
		}
		list = append(list, p)
	}
	// A broken stream ends the loop exactly like a complete one; without this
	// a truncated list goes out under a 200.
	if err := rows.Err(); err != nil {
		nexus.Error(w, http.StatusInternalServerError, "scan error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (m *Module) createProductHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := nexus.UserFromContext(r.Context())
	if err != nil {
		nexus.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		SKU    string  `json:"sku"`
		Name   string  `json:"name"`
		Price  float64 `json:"price"`
		Active bool    `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SKU == "" || req.Name == "" {
		nexus.Error(w, http.StatusBadRequest, "invalid product payload, sku and name required")
		return
	}

	id := uuid.New().String()
	now := time.Now()

	_, err = m.db.Exec(r.Context(),
		`INSERT INTO products (id, tenant_id, sku, name, price, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, claims.TenantID, req.SKU, req.Name, req.Price, req.Active, now, now)
	if err != nil {
		nexus.Error(w, http.StatusConflict, "sku already exists or DB error")
		return
	}

	p := Product{
		ID:        id,
		TenantID:  claims.TenantID,
		SKU:       req.SKU,
		Name:      req.Name,
		Price:     req.Price,
		Active:    req.Active,
		CreatedAt: now,
		UpdatedAt: now,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(p)
}

func (m *Module) updateProductHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := nexus.UserFromContext(r.Context())
	if err != nil {
		nexus.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	var req struct {
		SKU    string  `json:"sku"`
		Name   string  `json:"name"`
		Price  float64 `json:"price"`
		Active bool    `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		nexus.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}

	now := time.Now()
	res, err := m.db.Exec(r.Context(),
		`UPDATE products SET sku = $1, name = $2, price = $3, active = $4, updated_at = $5
		 WHERE id = $6 AND tenant_id = $7`,
		req.SKU, req.Name, req.Price, req.Active, now, id, claims.TenantID)
	if err != nil || res.RowsAffected() == 0 {
		nexus.Error(w, http.StatusNotFound, "product not found or unauthorized")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
