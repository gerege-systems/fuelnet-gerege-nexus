package settings

import (
	"testing"
)

// The registry's guarantees, which are the ones the rest of the platform
// depends on: a secret cannot be declared, a default that does not validate is
// caught at startup rather than in production, and an unregistered key answers
// with nothing rather than with something.

func TestASecretCannotBeRegistered(t *testing.T) {
	for _, key := range []string{
		"gemini.api_key", "sso.client_secret", "smtp.password",
		"integration.token", "signing.private_key",
	} {
		t.Run(key, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("%q was accepted as a setting", key)
				}
			}()
			Register(Spec{Key: key, Kind: KindString, Default: ""})
		})
	}
}

func TestAnInvalidDefaultIsRefusedAtStartup(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("a duration setting with a nonsense default was accepted")
		}
	}()
	Register(Spec{Key: "test.bad_default", Kind: KindDuration, Default: "soon"})
}

func TestValidateFollowsTheKind(t *testing.T) {
	cases := []struct {
		name  string
		spec  Spec
		value string
		ok    bool
	}{
		{"a duration", Spec{Kind: KindDuration}, "45m", true},
		{"not a duration", Spec{Kind: KindDuration}, "45 minutes", false},
		{"a negative duration", Spec{Kind: KindDuration}, "-1h", false},
		{"a bool", Spec{Kind: KindBool}, "true", true},
		{"not a bool", Spec{Kind: KindBool}, "yes please", false},
		{"an int in range", Spec{Kind: KindInt, Min: 1, Max: 10}, "5", true},
		{"an int out of range", Spec{Kind: KindInt, Min: 1, Max: 10}, "50", false},
		{"an enum option", Spec{Kind: KindEnum, Options: []string{"public", "private"}}, "public", true},
		{"not an enum option", Spec{Kind: KindEnum, Options: []string{"public", "private"}}, "sometimes", false},
		{"any string", Spec{Kind: KindString}, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.spec.Validate(c.value)
			if (err == nil) != c.ok {
				t.Fatalf("Validate(%q) = %v, want ok=%v", c.value, err, c.ok)
			}
		})
	}
}

// The access mode is private unless somebody has said otherwise, and that has
// to be true of a process with no database as well: the default is what a
// half-configured deployment gets, and the safe half-configured state is the
// closed one.
func TestTheAccessModeIsPrivateByDefault(t *testing.T) {
	if got := Get(AccessMode); got != AccessPrivate {
		t.Fatalf("the default access mode is %q, want %q", got, AccessPrivate)
	}
}

// The environment stays the fallback, so a deployment that has never opened the
// console behaves exactly as it did before this package existed.
func TestTheEnvironmentIsStillRead(t *testing.T) {
	t.Setenv("SESSION_IDLE_TIMEOUT", "45m")
	if got := Get(SessionIdleTimeout); got != "45m" {
		t.Fatalf("the environment value was ignored: %q", got)
	}
	if got := Duration(SessionIdleTimeout).Minutes(); got != 45 {
		t.Fatalf("Duration read %v minutes", got)
	}

	// And a value the environment holds that the registry would refuse falls
	// back to the default rather than being obeyed.
	t.Setenv("SESSION_IDLE_TIMEOUT", "half an hour")
	if got := Get(SessionIdleTimeout); got != "90m" {
		t.Fatalf("a nonsense environment value was used: %q", got)
	}
}

func TestAnUnregisteredKeyAnswersWithNothing(t *testing.T) {
	if got := Get("platform.does_not_exist"); got != "" {
		t.Fatalf("an unregistered key answered %q", got)
	}
}
