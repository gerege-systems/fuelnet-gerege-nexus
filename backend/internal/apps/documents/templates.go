package documents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/tenant"
)

// ErrTemplateNotFound is returned for a template id this tenant does not hold.
var ErrTemplateNotFound = errors.New("template not found")

// ErrTemplateNameTaken is returned when a tenant already has a template under
// the requested name.
var ErrTemplateNameTaken = errors.New("a template with this name already exists")

// DocumentTemplate is a preset a clerk starts a document from. A document is a
// title and a type, so a template carries exactly those two things — there is no
// document body for it to fill in.
type DocumentTemplate struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	DocType      string    `json:"doc_type"`
	TitlePattern string    `json:"title_pattern"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
}

const templateColumns = `id, name, doc_type, title_pattern, active, created_at`

func scanTemplate(row rowScanner) (*DocumentTemplate, error) {
	var tpl DocumentTemplate
	if err := row.Scan(&tpl.ID, &tpl.Name, &tpl.DocType, &tpl.TitlePattern, &tpl.Active, &tpl.CreatedAt); err != nil {
		return nil, err
	}
	return &tpl, nil
}

// resolveTitlePattern fills the placeholders a template may carry. Anything it
// does not recognise is left alone, so an unknown token shows up in the created
// document instead of vanishing silently.
func resolveTitlePattern(pattern string, now time.Time) string {
	replacements := []string{
		"{year}", strconv.Itoa(now.Year()),
		"{month}", fmt.Sprintf("%02d", int(now.Month())),
		"{date}", now.Format("2006-01-02"),
	}
	return strings.NewReplacer(replacements...).Replace(pattern)
}

func (m *DocumentsModule) ListTemplates(ctx context.Context, tenantID string) ([]DocumentTemplate, error) {
	rows, err := m.db.Query(ctx,
		`SELECT `+templateColumns+` FROM document_templates
		  WHERE tenant_id = $1 ORDER BY active DESC, name`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query templates: %w", err)
	}
	defer rows.Close()

	list := make([]DocumentTemplate, 0)
	for rows.Next() {
		tpl, err := scanTemplate(rows)
		if err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		list = append(list, *tpl)
	}
	return list, rows.Err()
}

func (m *DocumentsModule) CreateTemplate(ctx context.Context, tenantID, name, docType, titlePattern string) (*DocumentTemplate, error) {
	name = strings.TrimSpace(name)
	titlePattern = strings.TrimSpace(titlePattern)
	docType = strings.ToUpper(strings.TrimSpace(docType))

	if name == "" {
		return nil, errors.New("template name cannot be empty")
	}
	if titlePattern == "" {
		return nil, errors.New("title pattern cannot be empty")
	}
	if docType == "" {
		docType = DocTypes[0]
	}
	if !slices.Contains(DocTypes, docType) {
		return nil, fmt.Errorf("invalid doc_type %q", docType)
	}

	tpl, err := scanTemplate(m.db.QueryRow(ctx,
		`INSERT INTO document_templates (tenant_id, name, doc_type, title_pattern)
		      VALUES ($1, $2, $3, $4)
		   RETURNING `+templateColumns,
		tenantID, name, docType, titlePattern))
	if isUniqueViolation(err) {
		return nil, ErrTemplateNameTaken
	}
	if err != nil {
		return nil, fmt.Errorf("insert template: %w", err)
	}
	return tpl, nil
}

func (m *DocumentsModule) DeleteTemplate(ctx context.Context, tenantID, templateID string) error {
	if uuid.Validate(templateID) != nil {
		return ErrTemplateNotFound
	}
	tag, err := m.db.Exec(ctx,
		`DELETE FROM document_templates WHERE id = $1 AND tenant_id = $2`, templateID, tenantID)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

// CreateDocumentFromTemplate resolves the template's title pattern and routes the
// result exactly as a hand-typed document: the template decides what is created,
// never whether the approval rules apply.
func (m *DocumentsModule) CreateDocumentFromTemplate(ctx context.Context, tenantID, templateID string) (*Document, error) {
	if uuid.Validate(templateID) != nil {
		return nil, ErrTemplateNotFound
	}

	tpl, err := scanTemplate(m.db.QueryRow(ctx,
		`SELECT `+templateColumns+` FROM document_templates
		  WHERE id = $1 AND tenant_id = $2 AND active`, templateID, tenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTemplateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load template: %w", err)
	}

	return m.CreateDocument(ctx, tenantID, resolveTitlePattern(tpl.TitlePattern, time.Now()), tpl.DocType)
}

func (m *DocumentsModule) listTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	list, err := m.ListTemplates(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch templates")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (m *DocumentsModule) createTemplateHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Name         string `json:"name"`
		DocType      string `json:"doc_type"`
		TitlePattern string `json:"title_pattern"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid template payload")
		return
	}

	tpl, err := m.CreateTemplate(r.Context(), tenantID, req.Name, req.DocType, req.TitlePattern)
	switch {
	case errors.Is(err, ErrTemplateNameTaken):
		writeError(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tpl)
}

func (m *DocumentsModule) deleteTemplateHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	switch err := m.DeleteTemplate(r.Context(), tenantID, chi.URLParam(r, "id")); {
	case errors.Is(err, ErrTemplateNotFound):
		writeError(w, http.StatusNotFound, err.Error())
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "failed to delete template")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *DocumentsModule) useTemplateHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	doc, err := m.CreateDocumentFromTemplate(r.Context(), tenantID, chi.URLParam(r, "id"))
	switch {
	case errors.Is(err, ErrTemplateNotFound):
		writeError(w, http.StatusNotFound, err.Error())
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "failed to create document from template")
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}
