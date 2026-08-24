package config

import "testing"

func TestBrandNameFallsBackToTheProductName(t *testing.T) {
	t.Setenv("BRAND_NAME", "")
	if got := BrandName(); got != "Gerege Nexus" {
		t.Fatalf("unset BRAND_NAME should read as the product's own name, got %q", got)
	}
}

func TestBrandNameHonoursTheDeployment(t *testing.T) {
	t.Setenv("BRAND_NAME", "Salus")
	if got := BrandName(); got != "Salus" {
		t.Fatalf("BrandName() = %q, want the configured name", got)
	}
}

// A name written with the padding an .env file collects is the same name. The
// value ends up in a sentence and in front of a citizen approving an eID
// request, and " Salus " renders as a typo in both.
func TestBrandNameIgnoresSurroundingSpace(t *testing.T) {
	t.Setenv("BRAND_NAME", "  Salus  ")
	if got := BrandName(); got != "Salus" {
		t.Fatalf("BrandName() = %q, want the name without its padding", got)
	}
	t.Setenv("BRAND_NAME", "   ")
	if got := BrandName(); got != "Gerege Nexus" {
		t.Fatalf("a blank BRAND_NAME is not a name; got %q", got)
	}
}
