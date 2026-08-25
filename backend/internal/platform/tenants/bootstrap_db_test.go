package tenants

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/kernel/security"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/operator/optest"
)

// What a deployment's first way in has to be: an administrator who can actually
// sign in — which means the three things hand-written SQL got wrong — and a
// door that closes behind itself.

func TestBootstrapMakesAnAdministratorWhoCanSignIn(t *testing.T) {
	pool := optest.Pool(t)
	ctx := context.Background()

	// The writing half inside a transaction that is rolled back, because the
	// shared test database already has organisations in it and the check
	// Bootstrap does first would (rightly) refuse. The statements, the trigger
	// that creates the role and the grant are all the real ones.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	hash, err := security.HashPassword("a-long-enough-password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	slug := "bootstrap-" + strings.ReplaceAll(t.Name(), "/", "-")
	slug = strings.ToLower(slug)
	tenantID, userID, err := bootstrapTx(ctx, tx, slug, "First Organisation",
		"first@example.mn", "First Person", hash)
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	var stored string
	if err := tx.QueryRow(ctx, `SELECT password_hash FROM platform.users WHERE id = $1::uuid`,
		userID).Scan(&stored); err != nil {
		t.Fatalf("read the account: %v", err)
	}
	if !security.CheckPasswordHash("a-long-enough-password", stored) {
		t.Fatal("the administrator cannot sign in with the password that was chosen")
	}

	// Administration is the role on the membership, not users.is_admin. A
	// bootstrap that set the column and skipped the grant would leave somebody
	// who can sign in and do nothing.
	var admin bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (
		   SELECT 1 FROM tenant.membership_roles mr
		     JOIN tenant.memberships m ON m.id = mr.membership_id
		     JOIN tenant.roles r ON r.id = mr.role_id
		    WHERE m.tenant_id = $1::uuid AND m.user_id = $2::uuid AND r.code = 'admin')`,
		tenantID, userID).Scan(&admin); err != nil {
		t.Fatalf("read the grant: %v", err)
	}
	if !admin {
		t.Fatal("the first person is not an administrator of the organisation they were given")
	}
}

func TestBootstrapRefusesADeploymentThatAlreadyHasAnOrganisation(t *testing.T) {
	pool := optest.Pool(t)
	optest.Tenant(t, pool)

	_, _, err := Bootstrap(context.Background(), pool, FirstTenant{
		Slug: "second-way-in", Name: "Second Way In",
		AdminEmail: "intruder@example.mn", Password: "a-long-enough-password",
	})
	if !errors.Is(err, ErrAlreadyProvisioned) {
		t.Fatalf("want ErrAlreadyProvisioned, got %v", err)
	}
}
