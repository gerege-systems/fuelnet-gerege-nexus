-- +goose Up

-- State services app (io.example.gov_services).
--
-- Modelled on the service delivery domain of gerege-systems/open-gerege-mn:
-- a service passport registry, citizen applications with a status timeline,
-- appointments, and an officer queue.

-- Service passport — the catalogue of services an organisation delivers.
CREATE TABLE IF NOT EXISTS gov_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    name_en VARCHAR(255) NOT NULL DEFAULT '',
    category VARCHAR(64) NOT NULL DEFAULT 'GENERAL',
    description TEXT NOT NULL DEFAULT '',
    -- Fee in tugrik; 0 means the service is free of charge.
    fee NUMERIC(15, 2) NOT NULL DEFAULT 0,
    duration_days INT NOT NULL DEFAULT 1,
    -- Documents the applicant must supply, e.g. {"Иргэний үнэмлэх"}.
    required_evidences TEXT[] NOT NULL DEFAULT '{}',
    -- Whether the service can be booked for an in-person visit.
    appointment_required BOOLEAN NOT NULL DEFAULT FALSE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, code)
);
CREATE INDEX IF NOT EXISTS idx_gov_services_tenant ON gov_services(tenant_id);

-- Citizen applications against a service passport.
CREATE TABLE IF NOT EXISTS gov_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES gov_services(id) ON DELETE RESTRICT,
    reference VARCHAR(32) NOT NULL,
    applicant_name VARCHAR(255) NOT NULL,
    applicant_reg VARCHAR(32) NOT NULL DEFAULT '',
    applicant_phone VARCHAR(32) NOT NULL DEFAULT '',
    -- SUBMITTED → IN_REVIEW → INFO_REQUESTED → APPROVED | REJECTED → COMPLETED,
    -- or CANCELLED by the applicant at any point before a decision.
    status VARCHAR(32) NOT NULL DEFAULT 'SUBMITTED',
    note TEXT NOT NULL DEFAULT '',
    assigned_officer VARCHAR(255) NOT NULL DEFAULT '',
    decision_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at TIMESTAMPTZ,
    UNIQUE (tenant_id, reference)
);
CREATE INDEX IF NOT EXISTS idx_gov_applications_tenant ON gov_applications(tenant_id);
CREATE INDEX IF NOT EXISTS idx_gov_applications_status ON gov_applications(tenant_id, status);

-- Append-only timeline behind every application.
CREATE TABLE IF NOT EXISTS gov_application_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES gov_applications(id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    actor VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_gov_events_application ON gov_application_events(application_id);

-- In-person visit slots.
CREATE TABLE IF NOT EXISTS gov_appointments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    application_id UUID REFERENCES gov_applications(id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES gov_services(id) ON DELETE RESTRICT,
    citizen_name VARCHAR(255) NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    location VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'BOOKED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_gov_appointments_tenant ON gov_appointments(tenant_id, scheduled_at);

INSERT INTO apps (id, slug, name, description, icon_url, category, visibility) VALUES
    ('io.example.gov_services', 'gov-services', 'State Services',
     'Service passport registry, citizen applications, appointments and the officer queue.',
     '/icons/gov_services.png', 'Public Service', 'public')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM apps WHERE id = 'io.example.gov_services';
DROP TABLE IF EXISTS gov_appointments;
DROP TABLE IF EXISTS gov_application_events;
DROP TABLE IF EXISTS gov_applications;
DROP TABLE IF EXISTS gov_services;
