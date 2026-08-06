package esign

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/appregistry"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/gerege"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/tenant"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Default signature placement, verified live against the eSign service on A4
// (595x842pt) documents: the box top-left sits at (80,216) with the image
// anchored to the box bottom, landing the stamp under the signer block on the
// last page.
const (
	defaultSigX      uint = 80
	defaultSigY      uint = 216
	defaultSigWidth  uint = 200
	defaultSigHeight uint = 56
	defaultSigText        = "Тоон гарын үсгээр баталгаажив."

	maxPDFBytes = 15 << 20 // 15 MB decoded PDF upload limit
)

type Document struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Title       string     `json:"title"`
	FileName    string     `json:"file_name"`
	Status      string     `json:"status"` // PENDING, SIGNED
	PageCount   uint       `json:"page_count"`
	SignerName  string     `json:"signer_name,omitempty"`
	SignerRegNo string     `json:"signer_reg_no,omitempty"`
	SignerPhone string     `json:"signer_phone,omitempty"`
	SignedAt    *time.Time `json:"signed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type SignatureLog struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	DocumentID string    `json:"document_id,omitempty"`
	RegNo      string    `json:"reg_no"`
	PhoneNo    string    `json:"phone_no"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Action     string    `json:"action"` // CERT_CHECK, SIGN
	CreatedAt  time.Time `json:"created_at"`
}

type Module struct {
	db    *pgxpool.Pool
	esign *gerege.EsignService
}

func New(db *pgxpool.Pool, esignSvc *gerege.EsignService) *Module {
	m := &Module{db: db, esign: esignSvc}
	appregistry.Register(m)
	return m
}

func (m *Module) ID() string      { return "io.example.esign" }
func (m *Module) Name() string    { return "PDF E-Sign (Тоон гарын үсэг)" }
func (m *Module) Version() string { return "1.0.0" }

func (m *Module) Dependencies() []internal.Dependency {
	return nil
}

func (m *Module) Permissions() []internal.PermissionDefinition {
	return []internal.PermissionDefinition{
		{Code: "esign.read", Name: "View E-Sign Documents", Description: "View uploaded and signed PDF documents"},
		{Code: "esign.sign", Name: "Sign Documents", Description: "Sign PDF documents with a digital signature"},
		{Code: "esign.manage", Name: "Manage E-Sign Documents", Description: "Upload and manage PDF documents"},
	}
}

func (m *Module) Menus() []internal.MenuDefinition {
	return []internal.MenuDefinition{
		{ID: "esign", ParentID: "operations", Label: "PDF E-Sign", Path: "/esign", Icon: "pen-tool", Order: 55, Labels: map[string]string{"mn": "PDF цахим гарын үсэг"}},
	}
}

func (m *Module) RegisterRoutes(r chi.Router, tenantAuthMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/esign", func(er chi.Router) {
		er.Use(tenantAuthMiddleware)
		er.Get("/documents", m.listDocumentsHandler)
		er.Post("/documents", m.uploadDocumentHandler)
		er.Get("/documents/{id}/download", m.downloadDocumentHandler)
		er.Post("/documents/{id}/sign", m.signDocumentHandler)
		er.Post("/cert/check", m.checkCertHandler)
		er.Get("/logs", m.listLogsHandler)
	})
}

func (m *Module) listDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	rows, err := m.db.Query(r.Context(),
		`SELECT id, tenant_id, title, file_name, status, page_count,
		        COALESCE(signer_name, ''), COALESCE(signer_reg_no, ''), COALESCE(signer_phone, ''),
		        signed_at, created_at
		 FROM esign_documents WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	list := make([]Document, 0)
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.TenantID, &d.Title, &d.FileName, &d.Status, &d.PageCount,
			&d.SignerName, &d.SignerRegNo, &d.SignerPhone, &d.SignedAt, &d.CreatedAt); err != nil {
			http.Error(w, `{"error":"scan error"}`, http.StatusInternalServerError)
			return
		}
		list = append(list, d)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (m *Module) uploadDocumentHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		Title     string `json:"title"`
		FileName  string `json:"file_name"`
		PdfBase64 string `json:"pdf_base64"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" || req.PdfBase64 == "" {
		http.Error(w, `{"error":"title and pdf_base64 are required"}`, http.StatusBadRequest)
		return
	}

	pdfData, err := base64.StdEncoding.DecodeString(req.PdfBase64)
	if err != nil {
		http.Error(w, `{"error":"pdf_base64 is not valid base64"}`, http.StatusBadRequest)
		return
	}
	if len(pdfData) > maxPDFBytes {
		http.Error(w, `{"error":"PDF exceeds the 15MB upload limit"}`, http.StatusBadRequest)
		return
	}
	if len(pdfData) < 5 || string(pdfData[:5]) != "%PDF-" {
		http.Error(w, `{"error":"file is not a valid PDF document"}`, http.StatusBadRequest)
		return
	}

	if req.FileName == "" {
		req.FileName = "document.pdf"
	}
	pageCount := gerege.PDFPageCount(pdfData)

	var doc Document
	err = m.db.QueryRow(r.Context(),
		`INSERT INTO esign_documents (tenant_id, title, file_name, page_count, original_pdf)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, tenant_id, title, file_name, status, page_count, created_at`,
		tenantID, req.Title, req.FileName, pageCount, pdfData).Scan(
		&doc.ID, &doc.TenantID, &doc.Title, &doc.FileName, &doc.Status, &doc.PageCount, &doc.CreatedAt)
	if err != nil {
		http.Error(w, `{"error":"failed to store document"}`, http.StatusInternalServerError)
		return
	}

	audit.Record(r.Context(), tenantID, "system", "esign.document_uploaded", "esign", map[string]any{
		"document_id": doc.ID,
		"title":       doc.Title,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(doc)
}

func (m *Module) downloadDocumentHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	column := "original_pdf"
	if r.URL.Query().Get("variant") == "signed" {
		column = "signed_pdf"
	}

	var pdfData []byte
	var fileName string
	err = m.db.QueryRow(r.Context(),
		`SELECT `+column+`, file_name FROM esign_documents WHERE id = $1 AND tenant_id = $2`,
		id, tenantID).Scan(&pdfData, &fileName)
	if err != nil || len(pdfData) == 0 {
		http.Error(w, `{"error":"document not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
	_, _ = w.Write(pdfData)
}

func (m *Module) checkCertHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		PhoneNo string `json:"phone_no"`
		CivilID string `json:"civil_id"`
		Data    string `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PhoneNo == "" || req.CivilID == "" {
		http.Error(w, `{"error":"phone_no and civil_id are required"}`, http.StatusBadRequest)
		return
	}

	cert, err := m.esign.CheckCertificate(r.Context(), gerege.EsignCertRequest{
		PhoneNo: req.PhoneNo,
		CivilID: req.CivilID,
		Data:    req.Data,
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	_, _ = m.db.Exec(r.Context(),
		`INSERT INTO esign_signature_logs (tenant_id, reg_no, phone_no, first_name, last_name, action)
		 VALUES ($1, $2, $3, $4, $5, 'CERT_CHECK')`,
		tenantID, req.CivilID, req.PhoneNo, cert.SubjectDN.GivenName, cert.SubjectDN.Surname)

	audit.Record(r.Context(), tenantID, "system", "esign.cert_checked", "esign", map[string]any{
		"civil_id": req.CivilID,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"is_valid":    cert.IsValid,
		"given_name":  cert.SubjectDN.GivenName,
		"surname":     cert.SubjectDN.Surname,
		"common_name": cert.SubjectDN.CommonName,
		"uid":         cert.SubjectDN.UID,
	})
}

func (m *Module) signDocumentHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	var req struct {
		PhoneNo          string `json:"phone_no"`
		SignerName       string `json:"signer_name"`
		SignerRegNo      string `json:"signer_reg_no"`
		SignatureImage64 string `json:"signature_image64"`
		SignatureText    string `json:"signature_text"`
		X                uint   `json:"x"`
		Y                uint   `json:"y"`
		Width            uint   `json:"width"`
		Height           uint   `json:"height"`
		PageNumber       uint   `json:"page_number"` // 0 = last page
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PhoneNo == "" || req.SignatureImage64 == "" {
		http.Error(w, `{"error":"phone_no and signature_image64 are required"}`, http.StatusBadRequest)
		return
	}

	sigImage, err := base64.StdEncoding.DecodeString(req.SignatureImage64)
	if err != nil {
		http.Error(w, `{"error":"signature_image64 is not valid base64"}`, http.StatusBadRequest)
		return
	}

	var pdfData []byte
	var status string
	err = m.db.QueryRow(r.Context(),
		`SELECT original_pdf, status FROM esign_documents WHERE id = $1 AND tenant_id = $2`,
		id, tenantID).Scan(&pdfData, &status)
	if err != nil {
		http.Error(w, `{"error":"document not found"}`, http.StatusNotFound)
		return
	}
	if status == "SIGNED" {
		http.Error(w, `{"error":"document is already signed"}`, http.StatusConflict)
		return
	}

	if req.X == 0 {
		req.X = defaultSigX
	}
	if req.Y == 0 {
		req.Y = defaultSigY
	}
	if req.Width == 0 {
		req.Width = defaultSigWidth
	}
	if req.Height == 0 {
		req.Height = defaultSigHeight
	}
	if req.SignatureText == "" {
		req.SignatureText = defaultSigText
	}
	pageCount := gerege.PDFPageCount(pdfData)
	if req.PageNumber == 0 || req.PageNumber > pageCount {
		req.PageNumber = pageCount
	}

	signedPdf64, err := m.esign.SignPDF(r.Context(), gerege.EsignDocSignRequest{
		MSISDN:           "976" + req.PhoneNo,
		X:                req.X,
		Y:                req.Y,
		Width:            req.Width,
		Height:           req.Height,
		PageNumber:       req.PageNumber,
		SignatureText:    req.SignatureText,
		Pdf64:            base64.StdEncoding.EncodeToString(pdfData),
		SignatureImage64: req.SignatureImage64,
		ExtraImages:      make([]gerege.EsignExtraImage, 0),
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadGateway)
		return
	}

	signedPdf, err := base64.StdEncoding.DecodeString(signedPdf64)
	if err != nil {
		http.Error(w, `{"error":"eSign service returned invalid PDF data"}`, http.StatusBadGateway)
		return
	}

	now := time.Now()
	_, err = m.db.Exec(r.Context(),
		`UPDATE esign_documents
		 SET status = 'SIGNED', signed_pdf = $1, signature_image = $2,
		     signer_name = $3, signer_reg_no = $4, signer_phone = $5, signed_at = $6
		 WHERE id = $7 AND tenant_id = $8`,
		signedPdf, sigImage, req.SignerName, req.SignerRegNo, req.PhoneNo, now, id, tenantID)
	if err != nil {
		http.Error(w, `{"error":"failed to persist signed document"}`, http.StatusInternalServerError)
		return
	}

	_, _ = m.db.Exec(r.Context(),
		`INSERT INTO esign_signature_logs (tenant_id, document_id, reg_no, phone_no, first_name, action)
		 VALUES ($1, $2, $3, $4, $5, 'SIGN')`,
		tenantID, id, req.SignerRegNo, req.PhoneNo, req.SignerName)

	audit.Record(r.Context(), tenantID, "system", "esign.document_signed", "esign", map[string]any{
		"document_id": id,
		"signer":      req.SignerName,
		"page_number": req.PageNumber,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":      "SIGNED",
		"document_id": id,
		"signed_at":   now,
		"page_number": req.PageNumber,
	})
}

func (m *Module) listLogsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	rows, err := m.db.Query(r.Context(),
		`SELECT id, tenant_id, COALESCE(document_id::text, ''), COALESCE(reg_no, ''), COALESCE(phone_no, ''),
		        COALESCE(first_name, ''), COALESCE(last_name, ''), action, created_at
		 FROM esign_signature_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	list := make([]SignatureLog, 0)
	for rows.Next() {
		var l SignatureLog
		if err := rows.Scan(&l.ID, &l.TenantID, &l.DocumentID, &l.RegNo, &l.PhoneNo,
			&l.FirstName, &l.LastName, &l.Action, &l.CreatedAt); err != nil {
			http.Error(w, `{"error":"scan error"}`, http.StatusInternalServerError)
			return
		}
		list = append(list, l)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}
