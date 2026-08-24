package config

import (
	"os"
	"strings"
)

// defaultBrandName is what this platform calls itself when nobody has said
// otherwise. It is a default rather than a constant used directly: a name
// written into the code is a name the image carries, and an image that carries
// its own name cannot be deployed under somebody else's.
const defaultBrandName = "Gerege Nexus"

// BrandName is the product name a deployment answers to.
//
// The same argument as SelfOrigin's, one level up. An address baked into a
// build ties the image to one host; a *name* baked into a build ties it to one
// customer, and Level 1 of the ecosystem strategy — one image, a different
// .env, a hundred deployments — needs neither to be true. So the name is read
// from the environment, and the value here is only what an unset variable
// means.
//
// BRAND_NAME is read by the browser app as well (frontend/lib/brand.ts). The
// two are halves of one setting: a deployment that renames only one of them has
// a sign-in screen and an eID prompt that disagree about which product the
// person is standing in front of.
//
// What the backend does with it is small and deliberate. This is an API, so
// almost nothing it emits is prose; the two places the name reaches a human are
// the message shown when linking a verified eID identity fails, and the relying
// party name eID Mongolia puts in front of the citizen when it asks them to
// approve a request. Everything else visible is the browser app's.
func BrandName() string {
	if name := strings.TrimSpace(os.Getenv("BRAND_NAME")); name != "" {
		return name
	}
	return defaultBrandName
}
