/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * Whose tables these are.
 *
 * db/migrations is the platform's schema and the only place a table could be
 * created, which is why 28 of its 108 tables belonged to apps that are not in
 * this repository: commerce, the government services workflow, point of sale
 * and the registry. Their tables could not follow them out, so every deployment
 * carried the schema of every app anybody ever wrote here.
 *
 * Migration 00075 dropped all twenty-eight, and 00077 dropped eleven more:
 * the nine document_* tables, `departments` and `organisation_people`, which
 * went with their apps to client-gerege-nexus. Those eleven are a different
 * case and a harder one. The twenty-eight were unreachable from here — this
 * repository served none of those apps' routes — while these were being read
 * by code in this tree until the commit that dropped them. What made that possible was
 * nexus.Migrations (Үе 3): a module brings its own schema, so
 * business-gerege-nexus declares commerce's five itself. What made it safe was
 * counting the routes first — the core served none of those apps' endpoints,
 * so nothing here could read the tables anyway.
 *
 * Sixty-nine remain, and this test is what keeps that true.
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
	"device_enrollment_codes": "devices",
	"device_telemetry":        "devices", "devices": "devices",
	"eid_sign_state":      "eID",
	"email_verifications": "email verification",
	"esign_batch_items":   "signing rail", "esign_batches": "signing rail",
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
}

var (
	createTable = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:public\.)?([a-z0-9_]+)`)
	dropTable   = regexp.MustCompile(`(?i)DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:public\.)?([a-z0-9_]+)`)
	// Comments are stripped first. These files explain themselves at length,
	// and a comment quoting `CREATE TABLE IF NOT EXISTS` was read as a table
	// called "if".
	sqlComment = regexp.MustCompile(`(?m)--.*$`)
)

// upSection is the half of a migration that runs going forward.
//
// Reading the whole file would count a Down section's DROP as a drop and its
// CREATE as a creation, which is the opposite of what either does on `up`.
func upSection(source string) string {
	code := sqlComment.ReplaceAllString(source, "")
	if idx := strings.Index(source, "-- +goose Down"); idx >= 0 {
		code = sqlComment.ReplaceAllString(source[:idx], "")
	}
	return code
}

func TestPlatformMigrationsOwnNoAppTable(t *testing.T) {
	files, err := filepath.Glob("*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("read the migration directory: %v", err)
	}

	// What the schema ends up with, not what its history mentions. An applied
	// migration cannot be rewritten, so the twenty-eight tables migration 00075
	// dropped are still created by 00003, 00006, 00007, 00038 and 00040 — and
	// are not in any deployment's schema. Replaying creations and drops in
	// order is how a text scan answers the question the database would.
	sort.Strings(files)
	found := map[string]string{}
	for _, file := range files {
		source, err := os.ReadFile(file) // #nosec G304 -- a glob of this directory
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		code := upSection(string(source))
		for _, match := range createTable.FindAllStringSubmatch(code, -1) {
			found[strings.ToLower(match[1])] = file
		}
		for _, match := range dropTable.FindAllStringSubmatch(code, -1) {
			delete(found, strings.ToLower(match[1]))
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

// The platform's Go code must not name a table it does not own.
//
// internal/apps/boundaries_test.go already stops the platform importing an app
// package; the compiler enforces that half. This is the other half, and it is
// the half nothing enforces: a table name inside a SQL string is exactly the
// same dependency, and it survives the app leaving. The query keeps compiling
// and keeps returning rows — right up until the deployment that never had the
// app runs it against a table that is not there.
//
// The names are the twenty-eight that migration 00075 dropped. They no longer
// exist in any deployment's schema, so a query naming one now fails outright
// rather than quietly answering zero — which is the better failure, and still
// not one to ship.
func TestPlatformSQLNamesNoAppTable(t *testing.T) {
	departed := []string{
		"billing_invoices", "contacts", "products", "stock_levels", "stock_movements", "warehouses",
		"gov_application_events", "gov_applications", "gov_appointments", "gov_delivery_outbox",
		"gov_org_units", "gov_routing_rules", "gov_services", "gov_tasks", "gov_unit_members",
		"gov_upstream_connectors", "gov_workflow_steps", "gov_workflow_transitions",
		"gov_workflow_versions", "gov_workflows",
		"pos_shifts",
		"store_app_texts", "store_apps", "store_catalog_snapshots", "store_external_registrations",
		"store_publishers", "store_registry_state", "store_review_events",
	}
	sort.Strings(departed)

	// No allowlist. There were four entries when this test was written — the
	// assistant's commerce queries, the till shift handlers and three test
	// fixtures — and each was a debt with an owner. All four are paid: Үе 4a
	// took the assistant's, and this change took the rest with the tables.
	// A new entry here would be a new debt, which is a thing to argue about in
	// a review rather than to add quietly.

	// A table name where SQL would put one. Comments are stripped first, the
	// same as in the migration scan above: these files explain themselves, and
	// a sentence saying which tables a query *used* to read is not a query.
	comments := regexp.MustCompile(`(?s)/\*.*?\*/|(?m)//.*$`)
	pattern := regexp.MustCompile(`(?i)\b(?:FROM|JOIN|INTO|UPDATE|DELETE\s+FROM)\s+(?:public\.)?(` +
		strings.Join(departed, "|") + `)\b`)

	root := filepath.Join("..", "..")
	var offences []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err //nolint:wrapcheck // walk errors are reported as they are
		}
		source, err := os.ReadFile(path) // #nosec G304 -- a walk of this repository
		if err != nil {
			return err //nolint:wrapcheck
		}
		rel := filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator)))
		code := comments.ReplaceAllString(string(source), "")
		seen := map[string]bool{}
		for _, match := range pattern.FindAllStringSubmatch(code, -1) {
			table := strings.ToLower(match[1])
			if seen[table] {
				continue
			}
			seen[table] = true
			offences = append(offences, rel+" queries "+table)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan the backend: %v", err)
	}

	sort.Strings(offences)
	for _, offence := range offences {
		t.Errorf(`%s

That table belonged to a module in another repository and migration 00075
dropped it. A SQL string naming it is the same dependency a Go import would be,
and the only difference is that no compiler reports it — which is why these
outlived their modules. There is no table behind this query on any deployment.`, offence)
	}
}
