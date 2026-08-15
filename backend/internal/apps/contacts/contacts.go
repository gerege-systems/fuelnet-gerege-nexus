package contacts

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Contact struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Company   string    `json:"company"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Module struct {
	db    nexus.DB
	perms nexus.PermissionStore
}

// New builds the contact register. It registers no module, and that is the
// whole of what changed when the two cards became one.
//
// This was `io.gerege.nexus.contacts`, a second app beside Organisation &
// People. The two were one subject split in half: who this organisation is made
// of, and who it deals with. An administrator installing one and not the other
// got half a directory, and nothing in the store explained which half.
//
// So the register moved inside the organisation module, which owns the app's
// identity, permissions and menu, and calls RegisterRoutes below. The table,
// the routes and the screen are unchanged.
func New(p nexus.Platform) *Module {
	return &Module{db: p.DB(), perms: p.Permissions()}
}

// The permissions these routes are checked against — the organisation app's,
// because that is the app these routes now belong to.
//
// They are asserted here, per route, rather than left to the platform. The
// platform's blanket gate reads the *registered module's* route prefix, and the
// module registered for these routes is now organisation, which declares no
// prefix because it gates itself. Mounting this behind it without saying so
// would have turned "contacts.manage required" into "any member of the tenant",
// silently, in a diff about menus.
const (
	permRead   = "organisation.read"
	permManage = "organisation.manage"
)

func (m *Module) RegisterRoutes(r chi.Router, tenantAuthMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/contacts", func(cr chi.Router) {
		cr.Use(tenantAuthMiddleware)
		read := nexus.RequirePermission(m.perms, permRead)
		manage := nexus.RequirePermission(m.perms, permManage)
		cr.With(read).Get("/", m.listContactsHandler)
		cr.With(manage).Post("/", m.createContactHandler)
		cr.With(manage).Put("/{id}", m.updateContactHandler)
	})
}

func (m *Module) listContactsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	rows, err := m.db.Query(r.Context(),
		`SELECT id, tenant_id, name, email, phone, company, active, created_at, updated_at 
		 FROM contacts WHERE tenant_id = $1 ORDER BY name ASC`, tenantID)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := make([]Contact, 0)
	for rows.Next() {
		var c Contact
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Email, &c.Phone, &c.Company, &c.Active, &c.CreatedAt, &c.UpdatedAt); err != nil {
			nexus.Error(w, http.StatusInternalServerError, "scan error")
			return
		}
		list = append(list, c)
	}
	// A stream that breaks partway through ends the loop the same way a
	// complete one does, so without this the caller receives a short list
	// under a 200 and has no way to tell it apart from the whole set.
	if err := rows.Err(); err != nil {
		nexus.Error(w, http.StatusInternalServerError, "scan error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (m *Module) createContactHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := nexus.UserFromContext(r.Context())
	if err != nil {
		nexus.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Phone   string `json:"phone"`
		Company string `json:"company"`
		Active  bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		nexus.Error(w, http.StatusBadRequest, "invalid contact payload, name is required")
		return
	}

	id := uuid.New().String()
	now := time.Now()

	_, err = m.db.Exec(r.Context(),
		`INSERT INTO contacts (id, tenant_id, name, email, phone, company, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, claims.TenantID, req.Name, req.Email, req.Phone, req.Company, req.Active, now, now)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "failed to create contact")
		return
	}

	contact := Contact{
		ID:        id,
		TenantID:  claims.TenantID,
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		Company:   req.Company,
		Active:    req.Active,
		CreatedAt: now,
		UpdatedAt: now,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(contact)
}

func (m *Module) updateContactHandler(w http.ResponseWriter, r *http.Request) {
	claims, err := nexus.UserFromContext(r.Context())
	if err != nil {
		nexus.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	var req struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Phone   string `json:"phone"`
		Company string `json:"company"`
		Active  bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		nexus.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}

	now := time.Now()
	res, err := m.db.Exec(r.Context(),
		`UPDATE contacts SET name = $1, email = $2, phone = $3, company = $4, active = $5, updated_at = $6
		 WHERE id = $7 AND tenant_id = $8`,
		req.Name, req.Email, req.Phone, req.Company, req.Active, now, id, claims.TenantID)
	if err != nil || res.RowsAffected() == 0 {
		nexus.Error(w, http.StatusNotFound, "contact not found or unauthorized")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
