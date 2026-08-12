// Package apps assembles the business-module runtime. The platform owns HTTP,
// sessions and infrastructure; it does not need to import every module or know
// its route table.
package apps

import (
	"context"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/appstore_registry"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/billing"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/contacts"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/core"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/developer_portal"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/documents"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/esign"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/gov_services"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/inventory"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/products"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/publisher_studio"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/store_review"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/eidmongolia"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/gerege"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/integration"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/ssoprovider"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BackgroundModule interface {
	StartHousekeeping(context.Context)
}

type Runtime struct {
	Background []BackgroundModule
}

func Bootstrap(db *pgxpool.Pool, integrations *integration.Manager, eidMN *eidmongolia.Service, sso *ssoprovider.SSOProvider) Runtime {
	// First, and not merely in order: core is what the others assume. It is the
	// organisation, the people in it and how it is arranged — the module Odoo
	// calls base and never lets you uninstall.
	core.New(db)
	contacts.New(db)
	products.New(db)
	inventory.New(db, false)
	billing.New(db)
	documents.New(db)
	gov_services.New(db, integrations)
	developer_portal.NewDeveloperPortalModule(sso)
	// The App Store's own registry. Present in every build and dormant in
	// almost all of them: without a signing key it publishes no catalogue, and
	// without the app installed its routes are unreachable anyway.
	appstore_registry.New(db)
	// The two surfaces over it. Separate modules because a publisher submits
	// and a reviewer publishes, and one permission covering both would make the
	// queue decorative.
	publisher_studio.New(db)
	store_review.New(db)
	esignModule := esign.New(db, gerege.NewEsignService(), eidMN, integrations)
	return Runtime{Background: []BackgroundModule{esignModule}}
}
