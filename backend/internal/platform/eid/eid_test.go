package eid_test

import (
	"context"
	"testing"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/eid"
)

func TestEIDServiceMockOAuth2Exchange(t *testing.T) {
	svc := eid.NewEIDService()

	identity, err := svc.ExchangeCode(context.Background(), "mock_oauth_code_123", "http://localhost:3000/callback")
	if err != nil {
		t.Fatalf("unexpected error during mock E-ID exchange: %v", err)
	}

	if identity.RegNumber != "AA90010111" {
		t.Errorf("expected RegNumber AA90010111, got %s", identity.RegNumber)
	}
	if identity.AuthMethod != eid.AuthMethodPKISignature {
		t.Errorf("expected AuthMethod PKI_DIGITAL_SIGNATURE")
	}
}

func TestEIDServiceAuthenticateWithMethod(t *testing.T) {
	svc := eid.NewEIDService()

	identity, err := svc.AuthenticateWithMethod(context.Background(), "AA90010111", "123456", eid.AuthMethodMobileOTP)
	if err != nil {
		t.Fatalf("unexpected error during OTP authentication: %v", err)
	}

	if identity.RegNumber != "AA90010111" {
		t.Errorf("expected RegNumber AA90010111, got %s", identity.RegNumber)
	}
	if identity.AuthMethod != eid.AuthMethodMobileOTP {
		t.Errorf("expected AuthMethod MOBILE_OTP")
	}
}
