/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

package organisation

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/security"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/tenant"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// Person is somebody as this organisation knows them.
//
// The identity is the user's and is shared across organisations; the job title,
// the department and whether they still work here belong to the membership,
// because all three can differ in the next tenant the same person belongs to.
type Person struct {
	MembershipID   string   `json:"membership_id"`
	UserID         string   `json:"user_id"`
	Name           string   `json:"name"`
	Email          string   `json:"email"`
	Phone          string   `json:"phone"`
	JobTitle       string   `json:"job_title"`
	DepartmentID   string   `json:"department_id,omitempty"`
	DepartmentName string   `json:"department_name,omitempty"`
	Active         bool     `json:"active"`
	IsAdmin        bool     `json:"is_admin"`
	Roles          []string `json:"roles"`
	JoinedAt       string   `json:"joined_at"`
	// Which organisation this membership is in. Only meaningful when the
	// session is reading across more than one, which is when the screen shows
	// it — a column repeating the same name on every row is noise.
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
}

func (m *Module) handleListPeople(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}

	// Across every organisation this session is active in, which is just the
	// one it is acting in unless somebody has asked for more. The row-level
	// policies allow exactly the same set, so this widens what is asked for
	// without widening what could be reached.
	rows, err := m.db.Query(r.Context(),
		`SELECT ms.id::text, u.id::text, u.name, u.email, u.phone, ms.job_title,
		        COALESCE(ms.department_id::text, ''), COALESCE(d.name, ''),
		        ms.active, u.is_admin, ms.created_at::text,
		        COALESCE(ARRAY_AGG(r.code) FILTER (WHERE r.code IS NOT NULL), '{}'),
		        ms.tenant_id::text, tn.name
		   FROM memberships ms
		   JOIN users u ON u.id = ms.user_id
		   JOIN tenants tn ON tn.id = ms.tenant_id
		   LEFT JOIN departments d ON d.id = ms.department_id
		   LEFT JOIN membership_roles mr ON mr.membership_id = ms.id
		   LEFT JOIN roles r ON r.id = mr.role_id
		  WHERE ms.tenant_id = ANY($1::uuid[])
		  GROUP BY ms.id, u.id, d.name, tn.name
		  ORDER BY tn.name, ms.active DESC, u.name`, tenant.Allowed(r.Context()))
	if err != nil {
		slog.Error("core: could not list people", "error", err, "tenant_id", tenantID)
		httpx.Error(w, http.StatusInternalServerError, "could not load the people")
		return
	}
	defer rows.Close()

	people := make([]Person, 0)
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.MembershipID, &p.UserID, &p.Name, &p.Email, &p.Phone,
			&p.JobTitle, &p.DepartmentID, &p.DepartmentName, &p.Active, &p.IsAdmin,
			&p.JoinedAt, &p.Roles, &p.TenantID, &p.TenantName); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "could not read the people")
			return
		}
		people = append(people, p)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not read the people")
		return
	}
	httpx.JSON(w, http.StatusOK, people)
}

// handleUpdatePerson edits what this organisation knows about somebody.
//
// Only the membership's own fields. The person's name, email and language are
// theirs — editable from their own preferences and nowhere else — because an
// administrator of one tenant renaming a user would rename them in every other
// tenant that person belongs to.
func (m *Module) handleUpdatePerson(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}
	membershipID := chi.URLParam(r, "id")

	var body struct {
		JobTitle     *string `json:"job_title"`
		DepartmentID *string `json:"department_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "malformed request body")
		return
	}

	// An empty department means "none", which is a value rather than an
	// omission — hence the two-step: absent leaves it alone, empty clears it.
	var department *string
	if body.DepartmentID != nil {
		if trimmed := strings.TrimSpace(*body.DepartmentID); trimmed != "" {
			department = &trimmed
		}
	}

	tag, err := m.db.Exec(r.Context(),
		`UPDATE memberships SET
		     job_title     = COALESCE($3, job_title),
		     department_id = CASE WHEN $4 THEN $5::uuid ELSE department_id END
		 WHERE id = $1 AND tenant_id = $2`,
		membershipID, tenantID, body.JobTitle, body.DepartmentID != nil, department)
	if isForeignKeyViolation(err) {
		// Same composite key as everywhere else: a department belonging to
		// another organisation is refused by the schema, not by a check here.
		httpx.Error(w, http.StatusBadRequest, "that department is not in your organisation")
		return
	}
	if err != nil {
		slog.Error("core: could not update a person", "error", err, "membership_id", membershipID)
		httpx.Error(w, http.StatusInternalServerError, "could not save the change")
		return
	}
	if tag.RowsAffected() == 0 {
		// Not "forbidden": whether a membership exists in another tenant is not
		// this caller's business.
		httpx.Error(w, http.StatusNotFound, "this person is not in your organisation")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// handleDeactivatePerson ends somebody's membership without erasing them.
//
// Deactivation rather than deletion, because a membership is referenced by
// everything the person did here — a signature, an approval, a request they
// processed. Removing the row would either take that history with it or leave
// it pointing at nothing.
func (m *Module) handleDeactivatePerson(w http.ResponseWriter, r *http.Request) {
	m.setPersonActive(w, r, false)
}

func (m *Module) handleReactivatePerson(w http.ResponseWriter, r *http.Request) {
	m.setPersonActive(w, r, true)
}

func (m *Module) setPersonActive(w http.ResponseWriter, r *http.Request, active bool) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}
	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	membershipID := chi.URLParam(r, "id")

	if !active {
		// Locking yourself out is a support ticket, so it is refused here
		// rather than regretted later.
		var userID string
		if err := m.db.QueryRow(r.Context(),
			`SELECT user_id::text FROM memberships WHERE id = $1 AND tenant_id = $2`,
			membershipID, tenantID).Scan(&userID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.Error(w, http.StatusNotFound, "this person is not in your organisation")
				return
			}
			httpx.Error(w, http.StatusInternalServerError, "could not read the membership")
			return
		}
		if userID == claims.UserID {
			httpx.Error(w, http.StatusBadRequest, "you cannot deactivate your own membership")
			return
		}

		// And neither is leaving an organisation with nobody who can administer
		// it: the last active administrator stays.
		var remaining int
		if err := m.db.QueryRow(r.Context(),
			`SELECT count(*) FROM memberships ms
			   JOIN users u ON u.id = ms.user_id
			  WHERE ms.tenant_id = $1 AND ms.active AND u.is_admin AND ms.id <> $2`,
			tenantID, membershipID).Scan(&remaining); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "could not check the administrators")
			return
		}
		var isAdmin bool
		if err := m.db.QueryRow(r.Context(),
			`SELECT u.is_admin FROM memberships ms JOIN users u ON u.id = ms.user_id
			  WHERE ms.id = $1`, membershipID).Scan(&isAdmin); err == nil && isAdmin && remaining == 0 {
			httpx.Error(w, http.StatusBadRequest,
				"this is the last administrator of the organisation; appoint another one first")
			return
		}
	}

	tag, err := m.db.Exec(r.Context(),
		`UPDATE memberships SET active = $3,
		        deactivated_at = CASE WHEN $3 THEN NULL ELSE NOW() END
		 WHERE id = $1 AND tenant_id = $2`, membershipID, tenantID, active)
	if err != nil {
		slog.Error("core: could not change a membership", "error", err, "membership_id", membershipID)
		httpx.Error(w, http.StatusInternalServerError, "could not save the change")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "this person is not in your organisation")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"active": active})
}

// --- departments ------------------------------------------------------------

// Department is one node of the organisation's structure.
type Department struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	ParentID    string `json:"parent_id,omitempty"`
	ManagerID   string `json:"manager_membership_id,omitempty"`
	ManagerName string `json:"manager_name,omitempty"`
	Active      bool   `json:"active"`
	PeopleCount int    `json:"people_count"`
	TenantID    string `json:"tenant_id"`
	TenantName  string `json:"tenant_name"`
}

func (m *Module) handleListDepartments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}

	rows, err := m.db.Query(r.Context(),
		`SELECT d.id::text, d.code, d.name, COALESCE(d.parent_id::text, ''),
		        COALESCE(d.manager_membership_id::text, ''), COALESCE(u.name, ''), d.active,
		        (SELECT count(*) FROM memberships ms WHERE ms.department_id = d.id AND ms.active),
		        d.tenant_id::text, tn.name
		   FROM departments d
		   JOIN tenants tn ON tn.id = d.tenant_id
		   LEFT JOIN memberships mgr ON mgr.id = d.manager_membership_id
		   LEFT JOIN users u ON u.id = mgr.user_id
		  WHERE d.tenant_id = ANY($1::uuid[])
		  ORDER BY tn.name, d.active DESC, d.name`, tenant.Allowed(r.Context()))
	if err != nil {
		slog.Error("core: could not list departments", "error", err, "tenant_id", tenantID)
		httpx.Error(w, http.StatusInternalServerError, "could not load the departments")
		return
	}
	defer rows.Close()

	list := make([]Department, 0)
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.Code, &d.Name, &d.ParentID, &d.ManagerID,
			&d.ManagerName, &d.Active, &d.PeopleCount, &d.TenantID, &d.TenantName); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "could not read the departments")
			return
		}
		list = append(list, d)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not read the departments")
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

type departmentBody struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id"`
	ManagerID *string `json:"manager_membership_id"`
}

func (m *Module) handleCreateDepartment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}

	var body departmentBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "malformed request body")
		return
	}
	body.Code = strings.TrimSpace(body.Code)
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" || !security.IsValidSlug(body.Code) {
		httpx.Error(w, http.StatusBadRequest,
			"a department needs a name and a code of lowercase letters, digits, - and _")
		return
	}

	var id string
	err := m.db.QueryRow(r.Context(),
		`INSERT INTO departments (tenant_id, code, name, parent_id, manager_membership_id)
		 VALUES ($1, $2, $3, $4::uuid, $5::uuid) RETURNING id::text`,
		tenantID, body.Code, body.Name, emptyToNil(body.ParentID), emptyToNil(body.ManagerID)).Scan(&id)
	if isUniqueViolation(err) {
		httpx.Error(w, http.StatusConflict, "a department with that code already exists")
		return
	}
	if isForeignKeyViolation(err) {
		// The composite foreign keys make this the schema's answer to a parent
		// or a manager from another tenant, rather than a check somebody has to
		// remember to write.
		httpx.Error(w, http.StatusBadRequest, "the parent department or manager is not in your organisation")
		return
	}
	if err != nil {
		slog.Error("core: could not create a department", "error", err, "tenant_id", tenantID)
		httpx.Error(w, http.StatusInternalServerError, "could not create the department")
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (m *Module) handleUpdateDepartment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")

	var body departmentBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "malformed request body")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		httpx.Error(w, http.StatusBadRequest, "a department needs a name")
		return
	}
	// A department cannot be its own parent, and cannot be moved under one of
	// its own descendants either. The schema refuses the first; the second is a
	// walk and no CHECK can see it. Leaving it to the screen was not enough —
	// the screen offers a tree it drew from the same data, and anything that
	// posts this endpoint directly could tie a knot that makes every reader
	// that follows parent_id loop for ever.
	if body.ParentID != nil && strings.TrimSpace(*body.ParentID) != "" {
		parent := strings.TrimSpace(*body.ParentID)
		if parent == id {
			httpx.Error(w, http.StatusBadRequest, "a department cannot report to itself")
			return
		}
		var descendant bool
		if err := m.db.QueryRow(r.Context(),
			`WITH RECURSIVE below AS (
			    SELECT id FROM departments WHERE parent_id = $1 AND tenant_id = $3
			    UNION ALL
			    SELECT d.id FROM departments d JOIN below b ON d.parent_id = b.id
			     WHERE d.tenant_id = $3
			 )
			 SELECT EXISTS (SELECT 1 FROM below WHERE id = $2::uuid)`,
			id, parent, tenantID).Scan(&descendant); err != nil {
			slog.Error("core: could not check the department tree", "error", err, "department_id", id)
			httpx.Error(w, http.StatusInternalServerError, "could not save the department")
			return
		}
		if descendant {
			httpx.Error(w, http.StatusConflict,
				"that unit is already below this one; the two would report to each other")
			return
		}
	}

	tag, err := m.db.Exec(r.Context(),
		`UPDATE departments SET name = $3, parent_id = $4::uuid,
		        manager_membership_id = $5::uuid, updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID, strings.TrimSpace(body.Name), emptyToNil(body.ParentID), emptyToNil(body.ManagerID))
	if isForeignKeyViolation(err) {
		httpx.Error(w, http.StatusBadRequest, "the parent department or manager is not in your organisation")
		return
	}
	if err != nil {
		slog.Error("core: could not update a department", "error", err, "department_id", id)
		httpx.Error(w, http.StatusInternalServerError, "could not save the department")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "department not found")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// handleArchiveDepartment retires a department without deleting it.
//
// People and history point at it. Archiving keeps those references readable and
// takes it out of every list that offers a choice — which is what somebody
// pressing "delete" on an empty-looking department actually wants.
func (m *Module) handleArchiveDepartment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}

	tag, err := m.db.Exec(r.Context(),
		`UPDATE departments SET active = FALSE, updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2`, chi.URLParam(r, "id"), tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not archive the department")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "department not found")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

// handleRestoreDepartment brings an archived department back.
//
// Archiving is reversible by design — it exists so that a unit which still has
// people and history behind it can leave the lists without taking them with it
// — and until now the only half that was reachable was the archiving. A screen
// that lists what it archived and cannot undo it is a delete with extra steps.
func (m *Module) handleRestoreDepartment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")

	// A unit cannot come back under a parent that is still archived: the tree
	// would draw it as a root, its own parent would be missing from every list
	// that offers one, and the next edit would silently reparent it.
	var parentArchived bool
	var parentName string
	err := m.db.QueryRow(r.Context(),
		`SELECT p.name, NOT p.active
		   FROM departments d JOIN departments p ON p.id = d.parent_id
		  WHERE d.id = $1 AND d.tenant_id = $2`, id, tenantID).Scan(&parentName, &parentArchived)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusInternalServerError, "could not read the department")
		return
	}
	if parentArchived {
		httpx.Error(w, http.StatusConflict,
			"the unit this one reports to is archived; restore "+parentName+" first")
		return
	}

	tag, err := m.db.Exec(r.Context(),
		`UPDATE departments SET active = TRUE, updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		slog.Error("core: could not restore a department", "error", err, "department_id", id)
		httpx.Error(w, http.StatusInternalServerError, "could not restore the department")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "department not found")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "restored"})
}

// handleDeleteDepartment removes a unit that never really existed.
//
// Archiving is for a unit that did: people worked in it, documents name it, and
// the row has to stay for those to keep meaning anything. This is for the other
// case — a typo, a duplicate, a structure sketched out and thought better of —
// and it is refused the moment anything at all points at the row.
//
// Which is why the refusal counts rather than just saying no: "3 people and 2
// units report to this one" tells the operator what to move, and "nothing does"
// would not have been a refusal at all.
func (m *Module) handleDeleteDepartment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")

	var people, children int
	if err := m.db.QueryRow(r.Context(),
		`SELECT (SELECT count(*) FROM memberships WHERE department_id = $1 AND tenant_id = $2),
		        (SELECT count(*) FROM departments  WHERE parent_id     = $1 AND tenant_id = $2)`,
		id, tenantID).Scan(&people, &children); err != nil {
		slog.Error("core: could not check what a department holds", "error", err, "department_id", id)
		httpx.Error(w, http.StatusInternalServerError, "could not check the department")
		return
	}
	if people > 0 || children > 0 {
		httpx.Error(w, http.StatusConflict, fmt.Sprintf(
			"this unit still has %d people and %d units under it; move them first, or archive it instead",
			people, children))
		return
	}

	tag, err := m.db.Exec(r.Context(),
		`DELETE FROM departments WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		slog.Error("core: could not delete a department", "error", err, "department_id", id)
		httpx.Error(w, http.StatusInternalServerError, "could not delete the department")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "department not found")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// emptyToNil turns an absent or blank identifier into SQL NULL, so "no parent"
// and "unchanged" do not both arrive as the empty string.
func emptyToNil(value *string) *string {
	if value == nil {
		return nil
	}
	if trimmed := strings.TrimSpace(*value); trimmed != "" {
		return &trimmed
	}
	return nil
}

func isUniqueViolation(err error) bool { return sqlState(err) == "23505" }

func isForeignKeyViolation(err error) bool { return sqlState(err) == "23503" }

func sqlState(err error) string {
	if err == nil {
		return ""
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState()
	}
	return ""
}
