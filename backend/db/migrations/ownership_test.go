/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Whose tables these are.
 *
 * db/migrations is the platform's schema and the only place a table can be
 * created, which is why 28 of the 108 tables here belong to apps that are not
 * in this repository: commerce went to business-gerege-nexus, the government
 * services workflow to gerege-gov, point of sale to pos-gerege-nexus and the
 * registry to appstore-gerege-mn. Their tables could not follow, so every
 * deployment carries the schema of every app anybody ever wrote here.
 *
 * nexus.Migrations is the way out — a module brings its own schema now. Moving
 * the 28 is a separate change with a data migration behind it. What these tests
 * do is stop the number growing while that is arranged.
 */

package migrations_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Every table the platform's migrations create, and who it is for.
//
// A new name here is not refused — the platform does grow tables. It is made
// visible: adding one means adding a line to this list, and that line is a
// sentence in a review saying "this belongs to the platform", which somebody
// can disagree with. Before, a CREATE TABLE in a 300-line migration was
// indistinguishable from every other line in it.
var platformTables = map[string]string{
	// -------------------------------------------------------- the platform's
	"access_change_events": "access control", "announcements": "control plane",
	"app_dependencies": "app store", "app_installations": "app store",
	"app_versions": "app store", "apps": "app store",
	"audit_events": "audit", "credential_grants": "access recovery",
	"departments": "organisation", "device_enrollment_codes": "devices",
	"device_telemetry": "devices", "devices": "devices",
	"document_approval_steps": "documents", "document_eid_sign_sessions": "documents",
	"document_files": "documents", "document_records": "documents",
	"document_retention_rules": "documents", "document_signature_policies": "documents",
	"document_signatures": "documents", "document_templates": "documents",
	"document_workflow_steps": "documents", "eid_sign_state": "eID",
	"email_verification_clients": "email verification", "email_verifications": "email verification",
	"esign_batch_items": "signing rail", "esign_batches": "signing rail",
	"esign_documents": "signing rail", "esign_settings": "signing rail",
	"esign_sign_sessions": "signing rail", "esign_signature_logs": "signing rail",
	"feature_flag_overrides": "control plane", "feature_flags": "control plane",
	"identity_binding_sessions": "identity", "installation_events": "app store",
	"integration_deliveries": "integrations", "integration_oauth_states": "integrations",
	"integrations": "integrations", "membership_roles": "access control",
	"memberships": "access control", "oauth2_access_tokens": "OAuth2 provider",
	"oauth2_authorization_codes": "OAuth2 provider", "oauth2_clients": "OAuth2 provider",
	"oauth2_signing_keys": "OAuth2 provider",
	"oauth2_consents":     "OAuth2 provider", "oauth2_tokens": "OAuth2 provider",
	"operator_accounts": "control plane", "operator_audit": "control plane",
	"operator_impersonations": "control plane", "operator_sessions": "control plane",
	"pending_approvals": "control plane", "permissions": "access control",
	"platform_backups": "control plane", "platform_settings": "control plane",
	"platform_settings_history": "control plane", "push_tokens": "devices",
	"report_grants": "report sharing", "report_schedules": "reports",
	"role_permissions": "access control", "roles": "access control",
	"sessions": "auth", "staff_pin_credentials": "devices",
	"store_app_versions": "app store", "tenant_profiles": "tenants",
	"tenant_quotas": "tenants", "tenants": "tenants",
	"urtuu_deliveries": "Өртөө", "urtuu_inbox": "Өртөө",
	"urtuu_numbers": "Өртөө", "urtuu_outbox": "Өртөө",
	"urtuu_peer_codes": "Өртөө", "urtuu_peers": "Өртөө",
	"urtuu_request_codes": "Өртөө", "urtuu_task_events": "Өртөө",
	"urtuu_tasks": "Өртөө", "usage_events": "usage",
	"user_eid_identities": "identity", "user_sso_identities": "identity",
	"users": "users", "ai_knowledge": "assistant", "ai_prompts": "assistant",

	// ------------------------------------------------------------- departed
	// Twenty-eight tables belonging to modules that are in other repositories.
	// They are listed so this test is green, not because they belong here. The
	// next change is moving each group out with its data; see
	// docs/CORE_BOUNDARY_PLAN.md §5 Үе 3.

	// TODO: departed — commerce, business-gerege-nexus (6)
	"billing_invoices": "departed: commerce", "contacts": "departed: commerce",
	"products": "departed: commerce", "stock_levels": "departed: commerce",
	"stock_movements": "departed: commerce", "warehouses": "departed: commerce",

	// TODO: departed — government services, gerege-gov (14)
	"gov_application_events": "departed: gov", "gov_applications": "departed: gov",
	"gov_appointments": "departed: gov", "gov_delivery_outbox": "departed: gov",
	"gov_org_units": "departed: gov", "gov_routing_rules": "departed: gov",
	"gov_services": "departed: gov", "gov_tasks": "departed: gov",
	"gov_unit_members": "departed: gov", "gov_upstream_connectors": "departed: gov",
	"gov_workflow_steps": "departed: gov", "gov_workflow_transitions": "departed: gov",
	"gov_workflow_versions": "departed: gov", "gov_workflows": "departed: gov",

	// TODO: departed — point of sale, pos-gerege-nexus (1)
	"pos_shifts": "departed: pos",

	// TODO: departed — the registry, appstore-gerege-mn (7). Not the same thing
	// as the store screens every deployment has: those read `apps`,
	// `app_installations` and `store_app_versions`, which are above. These are
	// the registry's own — who publishes, what is under review, what the last
	// catalogue build produced.
	"store_app_texts": "departed: registry", "store_apps": "departed: registry",
	"store_catalog_snapshots": "departed: registry", "store_external_registrations": "departed: registry",
	"store_publishers": "departed: registry", "store_registry_state": "departed: registry",
	"store_review_events": "departed: registry",
}

var (
	createTable = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:public\.)?([a-z0-9_]+)`)
	// Comments are stripped first. These files explain themselves at length,
	// and a comment quoting `CREATE TABLE IF NOT EXISTS` was read as a table
	// called "if".
	sqlComment = regexp.MustCompile(`(?m)--.*$`)
)

func TestPlatformMigrationsOwnNoAppTable(t *testing.T) {
	files, err := filepath.Glob("*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("read the migration directory: %v", err)
	}

	found := map[string]string{}
	for _, file := range files {
		source, err := os.ReadFile(file) // #nosec G304 -- a glob of this directory
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		code := sqlComment.ReplaceAllString(string(source), "")
		for _, match := range createTable.FindAllStringSubmatch(code, -1) {
			found[strings.ToLower(match[1])] = file
		}
	}

	var unlisted []string
	for table, file := range found {
		if _, ok := platformTables[table]; !ok {
			unlisted = append(unlisted, table+" ("+file+")")
		}
	}
	sort.Strings(unlisted)
	if len(unlisted) > 0 {
		t.Errorf(`db/migrations creates %d table(s) the platform has not claimed:

	%s

An app's table belongs in the app's own migrations — a module registers them
with nexus.Migrations and they run under goose_db_version_<slug>. A table
created here is a table every deployment carries whether or not it has the app,
and it can never leave with the app: 28 of the 108 tables here are already in
that position.

If this really is a platform table, add it to platformTables in
db/migrations/ownership_test.go with a word for what it is for. That line is the
decision, and a review can disagree with it.`,
			len(unlisted), strings.Join(unlisted, "\n\t"))
	}

	// The list must not outlive the tables either: a name left here after its
	// migration was rewritten is a name the next table can quietly inherit.
	var stale []string
	for table := range platformTables {
		if _, ok := found[table]; !ok {
			stale = append(stale, table)
		}
	}
	sort.Strings(stale)
	for _, table := range stale {
		t.Errorf("platformTables claims %q but no migration creates it", table)
	}
}

// The platform's Go code must not name an app's table either.
//
// internal/apps/boundaries_test.go already stops the platform importing an app
// package; the compiler enforces that half. This is the other half, and it is
// the half nothing enforces: a table name inside a SQL string is exactly the
// same dependency, and it survives the app leaving. The query keeps compiling
// and keeps returning rows — right up until the deployment that never had the
// app runs it against a table that is not there, or worse, one that is there
// and empty.
//
// The names checked are the ones belonging to modules in other repositories,
// matched only where SQL would put a table: after FROM, JOIN, INTO, UPDATE or
// DELETE FROM. A prose mention of "products" in a comment is not a dependency.
func TestPlatformSQLNamesNoAppTable(t *testing.T) {
	departed := []string{}
	for table, owner := range platformTables {
		if strings.HasPrefix(owner, "departed:") {
			departed = append(departed, table)
		}
	}
	sort.Strings(departed)

	// What is allowed to name them, and why. Each entry is a debt with an owner,
	// not an exemption: the change that moves the table is the change that
	// deletes the line. A prefix matches a package or a single file.
	allowed := map[string]string{
		// Two LLM tools — erp_summary and search_products — over products,
		// contacts, warehouses and stock_levels. Commerce left; the tables are
		// still created here, so nothing errors. The assistant answers
		// "0 products, 0 customers" instead, confidently, on every deployment
		// that never had commerce. See docs/CORE_BOUNDARY_PLAN.md §3.4(A).
		//
		// TODO: removed by Үе 4a, which is the change that deletes this entry.
		"internal/platform/ai": "Үе 4a",

		// The shift endpoints under /devices/shifts. Point of sale went to
		// pos-gerege-nexus and these did not follow — the same arrangement the
		// frontend is in, where lib/api/_departed/shifts.ts still serves the
		// screens. Found by this test rather than by anybody noticing.
		//
		// TODO: moves with the point-of-sale extraction.
		"internal/platform/native_operations_handlers.go": "pos extraction",

		// Test fixtures. Each of these needs *some* tenant-scoped table to
		// prove a property of the platform with — that a query without a
		// WHERE tenant_id sees only its own rows, that an operator's console
		// cannot read a tenant's data, that a report which forgets the tenant
		// clause returns nothing — and each reached for a commerce table
		// because one was there. The properties are the platform's; the tables
		// they are demonstrated on are not.
		//
		// TODO: rewritten against a platform table when commerce's schema moves.
		"internal/platform/controlplane/controlplane_db_test.go": "commerce schema move",
		"internal/platform/dbguard/dbguard_test.go":              "commerce schema move",
		"internal/platform/reporting/engine_integration_test.go": "commerce schema move",
	}

	// A table name where SQL would put one.
	pattern := regexp.MustCompile(`(?i)\b(?:FROM|JOIN|INTO|UPDATE|DELETE\s+FROM)\s+(?:public\.)?(` +
		strings.Join(departed, "|") + `)\b`)

	root := filepath.Join("..", "..", "internal", "platform")
	var offences []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err //nolint:wrapcheck // walk errors are reported as they are
		}
		source, err := os.ReadFile(path) // #nosec G304 -- a walk of this repository
		if err != nil {
			return err //nolint:wrapcheck
		}
		rel := filepath.ToSlash(strings.TrimPrefix(path, filepath.Join("..", "..")+string(filepath.Separator)))
		for prefix := range allowed {
			if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
				return nil
			}
		}
		seen := map[string]bool{}
		for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
			table := strings.ToLower(match[1])
			if seen[table] {
				continue
			}
			seen[table] = true
			offences = append(offences, rel+" queries "+table+" ("+platformTables[table]+")")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan internal/platform: %v", err)
	}

	sort.Strings(offences)
	for _, offence := range offences {
		t.Errorf(`%s

That table belongs to a module in another repository. A SQL string naming it is
the same dependency a Go import would be, and the only difference is that no
compiler reports it — which is why it outlived the module. On a deployment that
never installed the app, the query either fails or, worse, succeeds against an
empty table and reports zero as an answer.`, offence)
	}
}
