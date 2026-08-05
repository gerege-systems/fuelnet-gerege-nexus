package eid_test

import (
	"context"
	"testing"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/eid"
)

func TestEIDServiceMockVerification(t *testing.T) {
	svc := eid.NewEIDService()

	identity, err := svc.VerifyToken(context.Background(), "eid_AA90010111")
	if err != nil {
		t.Fatalf("unexpected error during mock E-ID verification: %v", err)
	}

	if identity.RegNumber != "AA90010111" {
		t.Errorf("expected RegNumber AA90010111, got %s", identity.RegNumber)
	}
	if !identity.VerifiedStatus {
		t.Errorf("expected verified_status = true")
	}
}

func TestEIDServiceEmptyToken(t *testing.T) {
	svc := eid.NewEIDService()

	_, err := svc.VerifyToken(context.Background(), "")
	if err == nil {
		t.Fatal("expected error on empty token")
	}
}
