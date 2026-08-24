/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * What the platform's own apps used to be called.
 *
 * An app id is a primary key in `apps`, a foreign key in three more tables, a
 * string inside a stored manifest, and the name a registry publishes under. A
 * migration fixes the first four on this deployment; it cannot fix the fifth,
 * because the registry is a different system on a different release cycle and a
 * catalogue signed last week is still a valid signature this week.
 *
 * So a renamed app is resolved rather than rejected: a catalogue that arrives
 * naming the old id is rewritten into the new one as it is parsed, before
 * anything downstream compares it against the compiled modules. The slug is
 * rewritten with it, because that is what a store URL and a manifest filename
 * are keyed by.
 *
 * DEPRECATED: remove in vNEXT, when every registry has republished.
 *
 * In kernel because both planes ask the question. The catalogue is the
 * platform's and it applies these as it parses; a tenant installing an app by
 * the name it used to have is the request path asking the same thing. Neither
 * is the other's, and the answer is a map with no table behind it, which is
 * what the floor is for.
 */

package appid

// renamedIDs maps a retired app id to the one it is now.
//
// DEPRECATED: remove in vNEXT.
var renamedIDs = map[string]string{
	"io.gerege.nexus.core":             "io.gerege.nexus.organisation",
	"io.gerege.nexus.developer_portal": "io.gerege.nexus.sso_clients",
}

// renamedSlugs maps a retired slug to the one it is now. Kept beside the ids
// rather than derived from them: a slug is not the last segment of an id by
// rule, only by habit, and gov-services already breaks the habit.
//
// DEPRECATED: remove in vNEXT.
var renamedSlugs = map[string]string{
	"core":             "organisation",
	"developer_portal": "sso-clients",
}

// ResolveAppID answers with the current id for an app, which for everything
// that has never been renamed is the id it was given.
func ResolveAppID(id string) string {
	if current, renamed := renamedIDs[id]; renamed {
		return current
	}
	return id
}

// ResolveAppSlug answers with the current slug for an app.
func ResolveAppSlug(slug string) string {
	if current, renamed := renamedSlugs[slug]; renamed {
		return current
	}
	return slug
}
