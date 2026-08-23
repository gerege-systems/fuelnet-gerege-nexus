/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package platform

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/cache"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// A distribution's module is built after the platform has published what a
// module may ask for.
//
// This is the ordering bug the e-Government move found. ExtraModules used to be
// called at the top of NewServer, before a single Provide — so a module that
// asked for the state registry in its constructor, the way every module in this
// repository does, got nothing. Nothing failed: nexus.Capability returns a zero
// value and an error, the module logged a warning and served a degraded screen,
// and the deployment had the rail all along.
//
// A golden route file cannot see this and neither can a build: the signature is
// identical either way. So the property is asserted directly — from inside an
// ExtraModules callback, which is the only place it can be observed.
func TestADistributionsModuleIsBuiltAfterTheCapabilitiesExist(t *testing.T) {
	dsn := os.Getenv("AUTH_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set AUTH_TEST_DATABASE_URL to a migrated test database to run this test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	t.Setenv("APP_CATALOG_URL", "")

	// Everything a module carried by another repository reaches for today. Each
	// is checked inside the callback, because that is the moment in question:
	// asking again afterwards would pass however NewServer is ordered.
	var missing []string
	check := func(name string, err error) {
		if err != nil {
			missing = append(missing, name+": "+err.Error())
		}
	}
	var called bool
	register := func(p nexus.Platform) {
		called = true
		if p == nil {
			t.Error("a distribution's module was handed a nil platform")
		}
		_, err := nexus.StateRegistryOf()
		check("nexus.StateRegistry", err)
		_, err = nexus.AuditHistory()
		check("nexus.AuditReader", err)
		_, err = nexus.People()
		check("nexus.Directory", err)
		_, err = nexus.Reports()
		check("nexus.ReportEngine", err)
		_, err = nexus.EID()
		check("nexus.EIDSigner", err)
		_, err = nexus.DAN()
		check("nexus.DANAuthenticator", err)
		_, err = nexus.Capability[nexus.Signer]()
		check("nexus.Signer", err)
		_, err = nexus.Capability[nexus.StateRails]()
		check("nexus.StateRails", err)
		_, err = nexus.Capability[nexus.RateLimiter]()
		check("nexus.RateLimiter", err)
		_, err = nexus.Capability[nexus.MeetingBooker]()
		check("nexus.MeetingBooker", err)
		_, err = nexus.Capability[nexus.Link]()
		check("nexus.Link", err)
		_, err = nexus.SigningRailsOf()
		check("nexus.SigningRails", err)
	}

	if _, err := NewServer(pool, filepath.FromSlash("../../../catalog/apps.json"),
		cache.NewBus(ctx, nil), register); err != nil {
		t.Fatalf("build the server: %v", err)
	}
	if !called {
		t.Fatal("the ExtraModules callback was never called")
	}
	for _, name := range missing {
		t.Errorf("a distribution's module was built before the platform provided %s", name)
	}
}
