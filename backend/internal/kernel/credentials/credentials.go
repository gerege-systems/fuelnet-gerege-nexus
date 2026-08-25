/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// Package credentials holds the keys a deployment reaches other systems with,
// so that setting one does not mean editing a file on a server.
//
// It is the deliberate other half of internal/kernel/settings. That package
// refuses to hold a secret — there is no secret Kind, and Register panics on a
// key that merely reads like one — and the reason it gives is sound: an
// interface that can edit a credential is an interface that can point this
// platform at somebody else's database. Nothing here weakens that. What is
// here instead is the same problem answered with the protections the answer
// actually needs, kept away from the settings table so that neither set of
// rules has to be loosened to accommodate the other:
//
//   - **Sealed at rest.** A value is AES-256-GCM before it reaches a column,
//     under the deployment's own key. A database backup is not a list of
//     somebody's API keys, and a deployment with no key configured cannot save
//     one at all rather than saving it in the clear.
//   - **Write-only.** No route returns a value, and this package offers no way
//     to ask for one over HTTP. What a screen may see is a name, a
//     description, whether it is set, where the value came from and the last
//     four characters — enough to tell two keys apart and to know a rotation
//     landed, and not enough to use.
//   - **Every name declared in Go.** As with settings, the table holds values
//     and never names: a row nothing registered is ignored, so writing to the
//     table directly cannot introduce a credential the code would then use.
//   - **The environment still wins by default.** A deployment that sets the
//     variable and never opens the console behaves exactly as it did. A value
//     written in the console takes precedence, because somebody chose it more
//     recently and more deliberately — the same rule settings follows.
package credentials

import (
	"sort"
	"strings"
)

// Spec declares one credential.
type Spec struct {
	// Name is what the console and the table call it: dotted, lower case.
	Name string `json:"name"`
	// Env is the variable it falls back to, and the one a deployment that
	// never opens the console keeps using.
	Env string `json:"env"`
	// Description is shown beside the field. Say what stops working without
	// it, because that is what the operator is trying to decide.
	Description string `json:"description"`
	// Docs is where the key is obtained, when there is somewhere to send
	// somebody. Empty for a credential this ecosystem issues by hand.
	Docs string `json:"docs,omitempty"`
}

var registry = map[string]Spec{}

// Register declares a credential. It panics on a mistake, because every call is
// a package variable or an init: a build that would silently ignore a
// credential is worse than one that will not start.
func Register(spec Spec) {
	name := strings.TrimSpace(spec.Name)
	switch {
	case name == "":
		panic("credentials: a credential needs a name")
	case registry[name].Name != "":
		panic("credentials: " + name + " is registered twice")
	case strings.TrimSpace(spec.Env) == "":
		panic("credentials: " + name + " needs the environment variable it falls back to")
	}
	registry[name] = spec
}

// Lookup returns a registered spec.
func Lookup(name string) (Spec, bool) {
	spec, known := registry[name]
	return spec, known
}

// All returns every registered credential, by name.
func All() []Spec {
	specs := make([]Spec, 0, len(registry))
	for _, spec := range registry {
		specs = append(specs, spec)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
	return specs
}

// hint is the last four characters of a value, which is what a screen may show.
//
// Four, and only of something long enough that four does not give it away: a
// six-character secret shown four characters at a time is a secret with two
// characters left to guess.
func hint(value string) string {
	runes := []rune(value)
	if len(runes) < 12 {
		return ""
	}
	return string(runes[len(runes)-4:])
}
