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

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/audit"
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
		return nil, fmt.Errorf("%w: template name cannot be empty", ErrInvalidConfiguration)
	}
	if titlePattern == "" {
		return nil, fmt.Errorf("%w: title pattern cannot be empty", ErrInvalidConfiguration)
	}
	if docType == "" {
		docType = DocTypes[0]
	}
	if !slices.Contains(DocTypes, docType) {
		return nil, fmt.Errorf("%w: invalid doc_type %q", ErrInvalidConfiguration, docType)
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

	audit.Record(ctx, tenantID, actorFor(ctx), "documents.template_created", tpl.ID, map[string]any{
		"name": tpl.Name, "doc_type": tpl.DocType,
	})

	return tpl, nil
}

// UpdateTemplate changes a template in place. A template that has produced
// documents cannot simply be replaced — the documents keep their titles — so
// editing one is about what it will produce next; `active` retires it from the list
// without deleting the record of what it was.
func (m *DocumentsModule) UpdateTemplate(ctx context.Context, tenantID, templateID, name, docType, titlePattern string, active bool) (*DocumentTemplate, error) {
	if uuid.Validate(templateID) != nil {
		return nil, ErrTemplateNotFound
	}

	name = strings.TrimSpace(name)
	titlePattern = strings.TrimSpace(titlePattern)
	docType = strings.ToUpper(strings.TrimSpace(docType))

	if name == "" {
		return nil, fmt.Errorf("%w: template name cannot be empty", ErrInvalidConfiguration)
	}
	if titlePattern == "" {
		return nil, fmt.Errorf("%w: title pattern cannot be empty", ErrInvalidConfiguration)
	}
	if !slices.Contains(DocTypes, docType) {
		return nil, fmt.Errorf("%w: invalid doc_type %q", ErrInvalidConfiguration, docType)
	}

	tpl, err := scanTemplate(m.db.QueryRow(ctx,
		`UPDATE document_templates
		    SET name = $1, doc_type = $2, title_pattern = $3, active = $4
		  WHERE id = $5 AND tenant_id = $6
		  RETURNING `+templateColumns,
		name, docType, titlePattern, active, templateID, tenantID))
	if isUniqueViolation(err) {
		return nil, ErrTemplateNameTaken
	}
	if isNoRows(err) {
		return nil, ErrTemplateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update template: %w", err)
	}

	audit.Record(ctx, tenantID, actorFor(ctx), "documents.template_changed", tpl.ID, map[string]any{
		"name": tpl.Name, "doc_type": tpl.DocType, "active": tpl.Active,
	})
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

	audit.Record(ctx, tenantID, actorFor(ctx), "documents.template_deleted", templateID, nil)

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
		writeWriteFailure(r.Context(), w, err, "failed to save template")
		return
	}
	writeJSON(w, http.StatusCreated, tpl)
}

func (m *DocumentsModule) updateTemplateHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Name         string `json:"name"`
		DocType      string `json:"doc_type"`
		TitlePattern string `json:"title_pattern"`
		Active       bool   `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid template payload")
		return
	}

	tpl, err := m.UpdateTemplate(r.Context(), tenantID, chi.URLParam(r, "id"),
		req.Name, req.DocType, req.TitlePattern, req.Active)
	switch {
	case errors.Is(err, ErrTemplateNotFound):
		writeError(w, http.StatusNotFound, err.Error())
		return
	case errors.Is(err, ErrTemplateNameTaken):
		writeError(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		writeWriteFailure(r.Context(), w, err, "failed to save template")
		return
	}
	writeJSON(w, http.StatusOK, tpl)
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
		// A pattern that resolves to an over-long title is the template's problem,
		// and the operator can only fix it if they are told which it is.
		writeWriteFailure(r.Context(), w, err, "failed to create document from template")
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}
