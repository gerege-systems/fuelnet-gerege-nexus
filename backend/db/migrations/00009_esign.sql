-- +goose Up

-- PDF E-Sign app (io.example.esign): tenant-scoped PDF documents signed with
-- Mongolian digital signatures (тоон гарын үсэг) via the Gerege eSign HSM.
CREATE TABLE IF NOT EXISTS esign_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    file_name VARCHAR(255) NOT NULL DEFAULT 'document.pdf',
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    page_count INT NOT NULL DEFAULT 1,
    original_pdf BYTEA NOT NULL,
    signed_pdf BYTEA,
    signature_image BYTEA,
    signer_name VARCHAR(255),
    signer_reg_no VARCHAR(64),
    signer_phone VARCHAR(32),
    signed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_esign_documents_tenant ON esign_documents(tenant_id);

-- Audit trail of certificate checks and signing events, mirroring the
-- signature log kept by the Gerege eSign integration.
CREATE TABLE IF NOT EXISTS esign_signature_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    document_id UUID REFERENCES esign_documents(id) ON DELETE SET NULL,
    reg_no VARCHAR(64),
    phone_no VARCHAR(32),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    action VARCHAR(32) NOT NULL DEFAULT 'CERT_CHECK',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_esign_signature_logs_tenant ON esign_signature_logs(tenant_id);

-- +goose Down
DROP TABLE IF EXISTS esign_signature_logs;
DROP TABLE IF EXISTS esign_documents;
