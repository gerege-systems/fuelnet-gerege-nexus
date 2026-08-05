-- +goose Up

-- Server-side sessions. Previously the "session token" was simply the user's
-- UUID, which is also returned by /api/v1/auth/me — anyone who learned a user
-- id could authenticate as that user. Tokens are now random 256-bit values and
-- only their SHA-256 digest is stored.
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash CHAR(64) UNIQUE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    auth_method VARCHAR(32) NOT NULL DEFAULT 'password',
    user_agent TEXT NOT NULL DEFAULT '',
    ip_address VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- Billing and Documents previously created their tables at boot via
-- `CREATE TABLE IF NOT EXISTS` with the error discarded, bypassing goose and
-- leaving schema state undefined.
CREATE TABLE IF NOT EXISTS billing_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_number VARCHAR(64) NOT NULL,
    contact_name VARCHAR(255) NOT NULL,
    amount NUMERIC(15, 2) NOT NULL,
    vat_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    ebarimt_status VARCHAR(32) NOT NULL DEFAULT 'CREATED',
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, invoice_number)
);
CREATE INDEX IF NOT EXISTS idx_billing_invoices_tenant ON billing_invoices(tenant_id);

CREATE TABLE IF NOT EXISTS document_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    doc_type VARCHAR(64) NOT NULL DEFAULT 'CONTRACT',
    status VARCHAR(32) NOT NULL DEFAULT 'DRAFT',
    signed_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_document_records_tenant ON document_records(tenant_id);

-- OAuth2 / OIDC client applications registered through the Developer Portal.
CREATE TABLE IF NOT EXISTS oauth2_clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    client_id VARCHAR(128) UNIQUE NOT NULL,
    client_secret_hash CHAR(64) NOT NULL,
    client_name VARCHAR(255) NOT NULL,
    redirect_uris TEXT[] NOT NULL DEFAULT '{}',
    grant_types TEXT[] NOT NULL DEFAULT '{}',
    scopes TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The app store catalog ships six apps but only three were ever inserted into
-- `apps`, so installing the other three failed on the app_id foreign key.
INSERT INTO apps (id, slug, name, description, icon_url, category, visibility) VALUES
    ('io.example.billing', 'billing', 'Public Billing & e-Barimt',
     'State fee invoicing, public service payments, and e-Barimt tax receipts.',
     '/icons/billing.png', 'Finance', 'public'),
    ('io.example.documents', 'documents', 'Digital Documents & Signatures',
     'Enterprise document routing, digital signatures, and approval workflows.',
     '/icons/documents.png', 'Productivity', 'public'),
    ('io.example.developer_portal', 'developer_portal', 'Developer Portal & OAuth2 SSO',
     'OAuth2 / OpenID Connect client application management and API credentials.',
     '/icons/developer_portal.png', 'Developer Tools', 'public')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM apps WHERE id IN ('io.example.billing', 'io.example.documents', 'io.example.developer_portal');
DROP TABLE IF EXISTS oauth2_clients;
DROP TABLE IF EXISTS document_records;
DROP TABLE IF EXISTS billing_invoices;
DROP TABLE IF EXISTS sessions;
