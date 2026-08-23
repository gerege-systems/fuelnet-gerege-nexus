/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package apps

import (
	"sort"
	"strings"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/reports"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/sso_clients"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/apps/urtuu"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// Every module in this binary, as nil pointers, for the same reason
// corePolicies uses them: Permissions() returns a literal and constructing six
// modules would drag in a database to ask each of them a constant.
//
// A wider list than corePolicies, which names only the modules that gate
// themselves. Every module grants permissions, so every module is here — and
// TestEveryModuleInThisRepositoryIsClassified beside it is what stops one being
// left out.
var everyModule = map[string]nexus.Module{
	"reports":     (*reports.Module)(nil),
	"sso_clients": (*sso_clients.SSOClientsModule)(nil),
	"urtuu":       (*urtuu.Module)(nil),
}

// Who every permission in this binary reaches when an app is installed.
//
// Written down twice on purpose, the same arrangement and for the same reason
// as corePolicies above: a permission reaching more people than intended does
// not look like a bug from the outside. The app installs, the pages open, and
// nothing turns red on its own. So the module answers, and this claims the
// answer, and the two have to agree.
//
// "suffix" means the module declares nothing and the deprecated grammar rule
// decides — `.read` to managers and users, `.manage` to managers. That is all
// of this table today, which is the honest picture: the mechanism landed, and
// converting each module to say what it means is separate work with a security
// review attached. What has changed is that saying it is now possible, from
// outside this repository as well as inside — documents was the one module that
// stated a grant (`documents.sign` to managers and users), and it states it
// from client-gerege-nexus now, which is the proof that it can be said out
// there.
var defaultGrants = map[string]string{
	"reports.view":     "suffix",
	"reports.schedule": "suffix",
	"reports.share":    "suffix",

	"sso_clients.read":   "suffix",
	"sso_clients.manage": "suffix",

	"urtuu.read":    "suffix",
	"urtuu.manage":  "suffix",
	"urtuu.process": "suffix",
}

func TestEveryPermissionSaysWhoItReaches(t *testing.T) {
	seen := map[string]bool{}
	for name, module := range everyModule {
		for _, perm := range module.Permissions() {
			seen[perm.Code] = true

			if err := perm.Validate(); err != nil {
				t.Errorf("%s: %v", name, err)
				continue
			}

			want, listed := defaultGrants[perm.Code]
			if !listed {
				t.Errorf(`%s declares the permission %q, which this table does not name.

Add it, with either "suffix" (the module says nothing and the deprecated
grammar rule decides) or the roles it declares. The line is the claim a
reviewer reads; a permission that appears without one is a permission whose
reach nobody stated.`, name, perm.Code)
				continue
			}

			got := "suffix"
			if perm.AdminOnly {
				got = "admin only"
			}
			if len(perm.DefaultRoles) > 0 {
				roles := append([]string(nil), perm.DefaultRoles...)
				sort.Strings(roles)
				got = strings.Join(roles, ",")
			}
			if got != want {
				t.Errorf("%s: %s reaches %q, recorded as %q", name, perm.Code, got, want)
			}
		}
	}

	for code := range defaultGrants {
		if !seen[code] {
			t.Errorf("the table names %q, which no module in this binary declares", code)
		}
	}
}
