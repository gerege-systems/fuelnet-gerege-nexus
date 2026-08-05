package platform

import "testing"

func TestRoleCodeValidation(t *testing.T) {
	valid := []string{"admin", "sales_manager", "inventory.read"}
	invalid := []string{"A", "Admin", " has-space", "x/owner", "-admin"}
	for _, v := range valid {
		if !roleCodePattern.MatchString(v) {
			t.Errorf("expected %q valid", v)
		}
	}
	for _, v := range invalid {
		if roleCodePattern.MatchString(v) {
			t.Errorf("expected %q invalid", v)
		}
	}
}

func TestAppRequestPermission(t *testing.T) {
	if got := appRequestPermission("io.example.contacts", "GET"); got != "contacts.read" {
		t.Fatalf("got %q", got)
	}
	if got := appRequestPermission("io.example.contacts", "POST"); got != "contacts.manage" {
		t.Fatalf("got %q", got)
	}
	if got := appRequestPermission("io.example.gov_services", "POST"); got != "" {
		t.Fatalf("government workflow must keep action-level checks, got %q", got)
	}
}
