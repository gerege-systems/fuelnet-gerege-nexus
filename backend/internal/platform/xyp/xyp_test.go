package xyp_test

import (
	"context"
	"testing"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/xyp"
)

func TestXYPMockCitizenQuery(t *testing.T) {
	svc := xyp.NewXYPService()

	info, err := svc.GetCitizenInfo(context.Background(), "AA90010111")
	if err != nil {
		t.Fatalf("unexpected error during mock XYP citizen query: %v", err)
	}

	if info.RegNumber != "AA90010111" {
		t.Errorf("expected RegNumber AA90010111, got %s", info.RegNumber)
	}
	if !info.Verified {
		t.Errorf("expected verified = true")
	}
}

func TestXYPMockCompanyQuery(t *testing.T) {
	svc := xyp.NewXYPService()

	company, err := svc.GetCompanyInfo(context.Background(), "5589412")
	if err != nil {
		t.Fatalf("unexpected error during mock XYP company query: %v", err)
	}

	if company.Name != "Гэрэгэ Системс ХХК" {
		t.Errorf("unexpected company name: %s", company.Name)
	}
	if !company.VatPayer {
		t.Errorf("expected vat_payer = true")
	}
}
