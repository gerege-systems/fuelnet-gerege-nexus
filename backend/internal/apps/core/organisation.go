/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

package core

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/auth"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/httpx"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/tenant"
)

// Organisation is what the tenant is, as opposed to what it is called.
//
// The name on the sidebar is in `tenants`; everything here is the version a
// document, an invoice or a government request has to carry — a legal name, a
// registration number, an address somebody could deliver to.
type Organisation struct {
	TenantID           string `json:"tenant_id"`
	Slug               string `json:"slug"`
	Name               string `json:"name"`
	LegalName          string `json:"legal_name"`
	RegistrationNumber string `json:"registration_number"`
	TaxNumber          string `json:"tax_number"`
	CountryCode        string `json:"country_code"`
	Province           string `json:"province"`
	District           string `json:"district"`
	Khoroo             string `json:"khoroo"`
	AddressLine        string `json:"address_line"`
	PostalCode         string `json:"postal_code"`
	Phone              string `json:"phone"`
	Email              string `json:"email"`
	Website            string `json:"website"`
	LogoURL            string `json:"logo_url"`
	Timezone           string `json:"timezone"`
	Locale             string `json:"locale"`
	Currency           string `json:"currency"`
}

const organisationColumns = `SELECT t.id::text, t.slug, t.name,
	p.legal_name, p.registration_number, p.tax_number, p.country_code,
	p.province, p.district, p.khoroo, p.address_line, p.postal_code,
	p.phone, p.email, p.website, p.logo_url, p.timezone, p.locale, p.currency
	FROM tenants t
	JOIN tenant_profiles p ON p.tenant_id = t.id
	WHERE t.id = $1`

func (m *Module) handleGetOrganisation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}

	var o Organisation
	err := m.db.QueryRow(r.Context(), organisationColumns, tenantID).Scan(
		&o.TenantID, &o.Slug, &o.Name, &o.LegalName, &o.RegistrationNumber, &o.TaxNumber,
		&o.CountryCode, &o.Province, &o.District, &o.Khoroo, &o.AddressLine, &o.PostalCode,
		&o.Phone, &o.Email, &o.Website, &o.LogoURL, &o.Timezone, &o.Locale, &o.Currency)
	if err != nil {
		// The profile row is created with the tenant and by the migration, so
		// its absence is a broken invariant rather than a missing page.
		slog.Error("core: could not read the organisation", "error", err, "tenant_id", tenantID)
		httpx.Error(w, http.StatusInternalServerError, "could not load the organisation")
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

func (m *Module) handleUpdateOrganisation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}

	var body struct {
		Name               *string `json:"name"`
		LegalName          *string `json:"legal_name"`
		RegistrationNumber *string `json:"registration_number"`
		TaxNumber          *string `json:"tax_number"`
		CountryCode        *string `json:"country_code"`
		Province           *string `json:"province"`
		District           *string `json:"district"`
		Khoroo             *string `json:"khoroo"`
		AddressLine        *string `json:"address_line"`
		PostalCode         *string `json:"postal_code"`
		Phone              *string `json:"phone"`
		Email              *string `json:"email"`
		Website            *string `json:"website"`
		LogoURL            *string `json:"logo_url"`
		Timezone           *string `json:"timezone"`
		Locale             *string `json:"locale"`
		Currency           *string `json:"currency"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "malformed request body")
		return
	}

	// Pointers rather than values, so a form that sends three fields changes
	// three fields. A struct of plain strings would blank everything the caller
	// happened not to mention — which is how a registration number disappears
	// because somebody edited a phone number.
	tx, err := m.db.Begin(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not save the organisation")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	if body.Name != nil {
		if strings.TrimSpace(*body.Name) == "" {
			httpx.Error(w, http.StatusBadRequest, "an organisation needs a name")
			return
		}
		if _, err := tx.Exec(r.Context(),
			`UPDATE tenants SET name = $1 WHERE id = $2`, strings.TrimSpace(*body.Name), tenantID); err != nil {
			slog.Error("core: could not rename the organisation", "error", err, "tenant_id", tenantID)
			httpx.Error(w, http.StatusInternalServerError, "could not save the organisation")
			return
		}
	}

	if _, err := tx.Exec(r.Context(),
		`UPDATE tenant_profiles SET
		     legal_name          = COALESCE($2, legal_name),
		     registration_number = COALESCE($3, registration_number),
		     tax_number          = COALESCE($4, tax_number),
		     country_code        = COALESCE($5, country_code),
		     province            = COALESCE($6, province),
		     district            = COALESCE($7, district),
		     khoroo              = COALESCE($8, khoroo),
		     address_line        = COALESCE($9, address_line),
		     postal_code         = COALESCE($10, postal_code),
		     phone               = COALESCE($11, phone),
		     email               = COALESCE($12, email),
		     website             = COALESCE($13, website),
		     logo_url            = COALESCE($14, logo_url),
		     timezone            = COALESCE($15, timezone),
		     locale              = COALESCE($16, locale),
		     currency            = COALESCE($17, currency),
		     updated_at          = NOW()
		 WHERE tenant_id = $1`,
		tenantID, body.LegalName, body.RegistrationNumber, body.TaxNumber, body.CountryCode,
		body.Province, body.District, body.Khoroo, body.AddressLine, body.PostalCode,
		body.Phone, body.Email, body.Website, body.LogoURL, body.Timezone, body.Locale,
		body.Currency); err != nil {
		slog.Error("core: could not save the organisation", "error", err, "tenant_id", tenantID)
		httpx.Error(w, http.StatusInternalServerError, "could not save the organisation")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not save the organisation")
		return
	}
	m.handleGetOrganisation(w, r)
}

// Preferences are the person's own, and follow them between organisations.
type Preferences struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Locale   string `json:"locale"`
	Timezone string `json:"timezone"`
	// What the organisation would use if the person expresses no preference.
	// Sent alongside so a screen can say "Mongolian (organisation default)"
	// rather than showing an empty selector.
	OrganisationLocale   string `json:"organisation_locale"`
	OrganisationTimezone string `json:"organisation_timezone"`
}

func (m *Module) handleGetPreferences(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.Require(w, r)
	if !ok {
		return
	}
	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var p Preferences
	if err := m.db.QueryRow(r.Context(),
		`SELECT u.name, u.email, u.phone, u.locale, u.timezone, tp.locale, tp.timezone
		   FROM users u, tenant_profiles tp
		  WHERE u.id = $1 AND tp.tenant_id = $2`, claims.UserID, tenantID).
		Scan(&p.Name, &p.Email, &p.Phone, &p.Locale, &p.Timezone,
			&p.OrganisationLocale, &p.OrganisationTimezone); err != nil {
		slog.Error("core: could not read preferences", "error", err, "user_id", claims.UserID)
		httpx.Error(w, http.StatusInternalServerError, "could not load your preferences")
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (m *Module) handleUpdatePreferences(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.UserFromContext(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		Name     *string `json:"name"`
		Phone    *string `json:"phone"`
		Locale   *string `json:"locale"`
		Timezone *string `json:"timezone"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "malformed request body")
		return
	}
	if body.Name != nil && strings.TrimSpace(*body.Name) == "" {
		httpx.Error(w, http.StatusBadRequest, "a name cannot be empty")
		return
	}
	// The email is deliberately not editable here. It is the login and the
	// address a verification link goes to, so changing it is a proof-of-address
	// flow rather than a text field — see emailverify.
	if _, err := m.db.Exec(r.Context(),
		`UPDATE users SET
		     name     = COALESCE($2, name),
		     phone    = COALESCE($3, phone),
		     locale   = COALESCE($4, locale),
		     timezone = COALESCE($5, timezone)
		 WHERE id = $1`,
		claims.UserID, body.Name, body.Phone, body.Locale, body.Timezone); err != nil {
		slog.Error("core: could not save preferences", "error", err, "user_id", claims.UserID)
		httpx.Error(w, http.StatusInternalServerError, "could not save your preferences")
		return
	}
	m.handleGetPreferences(w, r)
}
