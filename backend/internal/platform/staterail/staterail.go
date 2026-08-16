/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

// What this deployment is wired to on the state's side, as a shape both halves
// can name.
//
// It lived in the egov app, and the platform imported the app to build the
// value it handed back — "egov names the shape, the platform answers it". That
// reads well and points the wrong way: the platform is what every deployment
// runs, an app is what one product ships, and an import from the first to the
// second pins the second into the core. A deployment that removes the
// e-Government app was compiling it anyway.
//
// So the shape moved to where the answer comes from. The clients that fill it
// in — gerege (ХУР), eid, dan — are platform packages beside this one, and an
// app that wants to show the wiring imports it the way it already imports them.
package staterail

// Rail is one connection to a state system, as this deployment is configured.
type Rail struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Mode is "live", "mock" or "unconfigured". Three states rather than a
	// boolean, because a mock rail answers every question and none of the
	// answers is authoritative — reporting that as "connected" is how a test
	// fixture ends up on a government form.
	Mode     string `json:"mode"`
	Endpoint string `json:"endpoint,omitempty"`
}

// Rails is how the platform tells a module what it is wired to. A function
// rather than a snapshot: the answer is read from the process's configuration,
// and reading it per request keeps the screen honest if that ever stops being
// fixed at boot.
type Rails func() []Rail
