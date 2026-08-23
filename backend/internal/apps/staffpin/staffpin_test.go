/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package staffpin

import (
	"context"
	"errors"
	"testing"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// The shape of a PIN, asserted where the PIN now lives.
//
// It was internal/platform's test until 2026-08-23 and moved with the code it
// is about: a format rule kept in a package that no longer enforces it is a
// test that passes whatever the enforcing package does.
func TestAPINIsFourToTwelveDigits(t *testing.T) {
	for _, pin := range []string{"0000", "123456", "123456789012"} {
		if !validPIN.MatchString(pin) {
			t.Errorf("valid PIN rejected: %s", pin)
		}
	}
	for _, pin := range []string{"123", "1234567890123", "12ab", "", "12 34"} {
		if validPIN.MatchString(pin) {
			t.Errorf("invalid PIN accepted: %q", pin)
		}
	}
}

// A secret that is not a PIN is refused before the database is asked.
//
// The reason is not tidiness: Verify is reached from a route that anybody
// standing at a till can call, and a secret of any length would otherwise be
// hashed against every active credential in the organisation before being
// rejected for its shape.
func TestAMalformedSecretIsRejectedWithoutReadingAnything(t *testing.T) {
	m := &Module{
		installed: func(context.Context, string) (map[string]bool, error) {
			t.Error("the installed-apps gate was consulted for a secret that is not a PIN")
			return nil, nil
		},
	}
	if _, err := m.Verify(context.Background(), "tenant", "not-a-pin"); !errors.Is(err, nexus.ErrStaffCredentialRejected) {
		t.Errorf("got %v, want ErrStaffCredentialRejected", err)
	}
}

// An organisation that has not installed this app authenticates nobody through
// it, and the refusal is indistinguishable from a wrong PIN.
//
// This is the per-tenant gate the app gate cannot apply: the sign-in route
// carries a device token and no session. If this check goes missing, every
// organisation on the deployment gets staff switching whether it installed the
// app or not — and nothing else would notice.
func TestAnOrganisationWithoutTheAppIsRefused(t *testing.T) {
	m := &Module{
		installed: func(context.Context, string) (map[string]bool, error) {
			return map[string]bool{"io.gerege.nexus.reports": true}, nil
		},
	}
	if _, err := m.Verify(context.Background(), "tenant", "123456"); !errors.Is(err, nexus.ErrStaffCredentialRejected) {
		t.Errorf("got %v, want ErrStaffCredentialRejected", err)
	}
}

// A gate that cannot answer is not a gate that said no.
//
// Verify's caller turns ErrStaffCredentialRejected into 401 and everything else
// into 503, so returning the sentinel here would have somebody retyping a
// correct PIN at a database outage.
func TestAGateThatCannotAnswerIsNotARejection(t *testing.T) {
	boom := errors.New("the database is down")
	m := &Module{
		installed: func(context.Context, string) (map[string]bool, error) { return nil, boom },
	}
	_, err := m.Verify(context.Background(), "tenant", "123456")
	if errors.Is(err, nexus.ErrStaffCredentialRejected) {
		t.Error("an unreadable gate was reported as a rejected PIN")
	}
	if !errors.Is(err, boom) {
		t.Errorf("got %v, want the underlying failure", err)
	}
}
