/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

// Package settings holds the platform's configuration that can change without
// a deployment.
//
// Not all of it, and the line is deliberate. A value belongs here when it
// answers "how should the platform behave" — a timeout, an interval, which
// model to ask, whether strangers may sign up. A value does not belong here
// when it answers "how does the platform reach something else", because those
// are addresses and credentials, and an interface that can edit a credential
// is an interface that can point this platform at somebody else's database.
//
// Three rules make that line hold rather than merely describe it:
//
//   - **There is no secret kind.** A Kind is one of five values, none of them
//     "secret", so a secret cannot be registered — not "should not be", cannot.
//     Register also refuses a key that reads like one, which catches the case
//     where somebody stores a token as a String.
//   - **Every key is declared in Go.** The database holds values, never keys:
//     a row whose key is not in this registry is ignored by Get, so writing
//     directly to platform_settings cannot introduce behaviour.
//   - **The environment is the fallback, not the loser.** A deployment that
//     has never touched the console behaves exactly as it did before this
//     package existed, because Get consults the database, then the environment
//     variable the spec names, then the default.
package settings

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Kind is what a value is, and there are five.
//
// The list is closed on purpose — see the package note. Adding "secret" here
// would be a one-line change and the whole reason this type exists is that it
// should not be a one-line change.
type Kind string

const (
	KindBool     Kind = "bool"
	KindInt      Kind = "int"
	KindDuration Kind = "duration"
	KindString   Kind = "string"
	KindEnum     Kind = "enum"
)

// Spec declares one setting.
//
// The JSON tags are here rather than in a MarshalJSON method on purpose: Value
// embeds Spec, and an embedded type with its own MarshalJSON takes over the
// whole struct — the outer type's fields silently disappear from the response.
type Spec struct {
	Key  string `json:"key"`
	Kind Kind   `json:"kind"`
	// Default is what the platform does when nobody has said otherwise. It is
	// a string for the same reason the column is: this is the value as an
	// operator would type it.
	Default string `json:"default"`
	// Env is the variable that keeps working. A deployment that sets it and
	// never opens the console sees no change in behaviour; a value written in
	// the console takes precedence over it, because somebody chose it more
	// recently and more deliberately.
	Env string `json:"env,omitempty"`
	// Options are the permitted values of an enum.
	Options []string `json:"options,omitempty"`
	// Min and Max bound an int. Both zero means unbounded.
	Min int `json:"min,omitempty"`
	Max int `json:"max,omitempty"`
	// Description is shown beside the field in the console. Write it for the
	// operator at three in the morning, not for the developer who added it.
	Description string `json:"description"`
}

// registry is every setting this build knows about.
var registry = map[string]Spec{}

// secretish is the guard against the mistake this package cannot express in
// the type system: registering a credential as a String.
//
// A name is a good enough signal in practice — nobody calls a timeout
// "api_key" — and the cost of a false positive is having to rename a setting,
// which is cheap and happens at startup rather than in production.
var secretish = []string{"secret", "password", "token", "api_key", "apikey", "private_key", "credential"}

// Register declares a setting. It panics on anything wrong, because every call
// is in an init function or a package variable: a mistake here is a mistake in
// the build, and a build that would silently ignore a setting is worse than
// one that will not start.
func Register(spec Spec) {
	key := strings.TrimSpace(spec.Key)
	switch {
	case key == "":
		panic("settings: a setting needs a key")
	case registry[key].Key != "":
		panic("settings: " + key + " is registered twice")
	}
	for _, word := range secretish {
		if strings.Contains(strings.ToLower(key), word) {
			panic("settings: " + key + " reads like a secret; secrets belong in the environment, " +
				"never in a table an operator can edit")
		}
	}
	switch spec.Kind {
	case KindBool, KindInt, KindDuration, KindString:
	case KindEnum:
		if len(spec.Options) == 0 {
			panic("settings: " + key + " is an enum with no options")
		}
	default:
		panic("settings: " + key + " has an unknown kind " + string(spec.Kind))
	}
	if err := spec.Validate(spec.Default); err != nil {
		panic("settings: " + key + " has an invalid default: " + err.Error())
	}
	registry[key] = spec
}

// Lookup returns a registered spec.
func Lookup(key string) (Spec, bool) {
	spec, known := registry[key]
	return spec, known
}

// All returns every registered setting, by key.
func All() []Spec {
	specs := make([]Spec, 0, len(registry))
	for _, spec := range registry {
		specs = append(specs, spec)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Key < specs[j].Key })
	return specs
}

// Validate reports whether a value is one this setting may hold.
//
// It is the same check on both sides — the console calls it before writing,
// and the store calls it when reading, so a value that reached the table some
// other way is ignored rather than obeyed.
func (s Spec) Validate(value string) error {
	switch s.Kind {
	case KindBool:
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("%q is not true or false", value)
		}
	case KindInt:
		number, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%q is not a whole number", value)
		}
		if s.Min != 0 || s.Max != 0 {
			if number < s.Min || number > s.Max {
				return fmt.Errorf("%d is outside %d…%d", number, s.Min, s.Max)
			}
		}
	case KindDuration:
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("%q is not a duration such as 45m or 2h", value)
		}
		if parsed < 0 {
			return fmt.Errorf("%q is negative", value)
		}
	case KindEnum:
		for _, option := range s.Options {
			if option == value {
				return nil
			}
		}
		return fmt.Errorf("%q is not one of %s", value, strings.Join(s.Options, ", "))
	case KindString:
		// Anything, including empty: a string setting whose empty value means
		// "unset" is a common and useful shape.
	}
	return nil
}
