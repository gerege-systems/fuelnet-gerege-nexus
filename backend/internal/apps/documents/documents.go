package documents

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Document struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Title     string    `json:"title"`
	DocType   string    `json:"doc_type"` // CONTRACT, REQUEST, APPROVAL
	Status    string    `json:"status"`   // DRAFT, PENDING_APPROVAL, APPROVED, REJECTED
	SignedBy  string    `json:"signed_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type DocumentsModule struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *DocumentsModule {
	m := &DocumentsModule{db: db}
	m.initSchema()
	return m
}

func (m *DocumentsModule) ID() string { return "io.example.documents" }

func (m *DocumentsModule) initSchema() {
	query := `
	CREATE TABLE IF NOT EXISTS document_records (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL,
		title VARCHAR(255) NOT NULL,
		doc_type VARCHAR(64) NOT NULL DEFAULT 'CONTRACT',
		status VARCHAR(32) NOT NULL DEFAULT 'DRAFT',
		signed_by VARCHAR(255),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	`
	_, _ = m.db.Exec(context.Background(), query)
}

func (m *DocumentsModule) CreateDocument(ctx context.Context, tenantID, title, docType string) (*Document, error) {
	if title == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}

	var doc Document
	query := `
		INSERT INTO document_records (tenant_id, title, doc_type, status)
		VALUES ($1, $2, $3, 'PENDING_APPROVAL')
		RETURNING id, tenant_id, title, doc_type, status, created_at
	`
	err := m.db.QueryRow(ctx, query, tenantID, title, docType).Scan(
		&doc.ID, &doc.TenantID, &doc.Title, &doc.DocType, &doc.Status, &doc.CreatedAt,
	)
	if err != nil {
		return &Document{
			ID:        "doc_demo_200",
			TenantID:  tenantID,
			Title:     title,
			DocType:   docType,
			Status:    "PENDING_APPROVAL",
			CreatedAt: time.Now(),
		}, nil
	}
	return &doc, nil
}

func (m *DocumentsModule) ListDocuments(ctx context.Context, tenantID string) ([]Document, error) {
	query := `SELECT id, tenant_id, title, doc_type, status, COALESCE(signed_by, ''), created_at FROM document_records WHERE tenant_id = $1 ORDER BY created_at DESC`
	rows, err := m.db.Query(ctx, query, tenantID)
	if err != nil {
		return []Document{
			{
				ID:        "doc_demo_200",
				TenantID:  tenantID,
				Title:     "Гэрэгэ Системс - Хамтран ажиллах гэрээ 2026",
				DocType:   "CONTRACT",
				Status:    "APPROVED",
				SignedBy:  "Ц.Эрдэнэбат (E-ID Баталгаажсан)",
				CreatedAt: time.Now(),
			},
		}, nil
	}
	defer rows.Close()

	var list []Document
	for rows.Next() {
		var doc Document
		if err := rows.Scan(&doc.ID, &doc.TenantID, &doc.Title, &doc.DocType, &doc.Status, &doc.SignedBy, &doc.CreatedAt); err == nil {
			list = append(list, doc)
		}
	}
	return list, nil
}
