/*
 * Gerege Nexus — App Store registry
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * catalog-sign turns a catalog directory into the signed document the registry
 * serves, and makes the key pair that signs it.
 *
 * Two jobs, both of which exist because the registry should not be the only way
 * to do them:
 *
 *   catalog-sign -genkey
 *       Makes an Ed25519 pair. The private half becomes the registry's
 *       SIGNING_KEY; the public half is pinned in each Nexus instance's
 *       APPSTORE_PUBLIC_KEY. Nothing else ever needs the private half — not
 *       even to verify.
 *
 *   catalog-sign -catalog catalog/apps.json -key <base64> -out catalog.json
 *       Signs a catalogue offline. That is how an air-gapped operator publishes
 *       to their own instances, and how this repository tests the client
 *       against bytes no registry produced.
 */

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/appstore"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
)

func main() {
	var (
		genKey      = flag.Bool("genkey", false, "generate an Ed25519 key pair and print both halves")
		catalogPath = flag.String("catalog", "catalog/apps.json", "path to apps.json (manifests are read beside it)")
		keyValue    = flag.String("key", "", "base64 Ed25519 private key; defaults to $SIGNING_KEY")
		keyID       = flag.String("key-id", "appstore-2026", "key id carried in the document")
		platform    = flag.String("platform", "", "platform version to validate manifests against; empty skips the check")
		generatedAt = flag.String("generated-at", "", "RFC3339 timestamp inside the signature; defaults to now")
		out         = flag.String("out", "-", "where to write the signed document, or - for stdout")
	)
	flag.Parse()

	if *genKey {
		public, private, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			fail("generate a key: %v", err)
		}
		fmt.Printf("SIGNING_KEY=%s\n", base64.StdEncoding.EncodeToString(private))
		fmt.Printf("APPSTORE_PUBLIC_KEY=%s\n", base64.StdEncoding.EncodeToString(public))
		return
	}

	key := *keyValue
	if key == "" {
		key = os.Getenv("SIGNING_KEY")
	}
	if key == "" {
		fail("no signing key: pass -key or set SIGNING_KEY (or use -genkey to make one)")
	}
	signer, err := appstore.NewSigner(*keyID, key)
	if err != nil {
		fail("%v", err)
	}

	catalog, err := appcatalog.LoadFile(*catalogPath, *platform)
	if err != nil {
		fail("read the catalog: %v", err)
	}

	// The same assembly the registry endpoint uses, so a document signed here
	// and one served there are the same kind of thing rather than two
	// implementations of one description.
	stamp := time.Now().UTC()
	if *generatedAt != "" {
		parsed, err := time.Parse(time.RFC3339, *generatedAt)
		if err != nil {
			fail("-generated-at must be RFC3339: %v", err)
		}
		stamp = parsed
	}

	document, _, err := appstore.SignDocument(signer, stamp, catalog)
	if err != nil {
		fail("encode the document: %v", err)
	}

	if *out == "-" {
		fmt.Println(string(document))
		return
	}
	if err := os.WriteFile(*out, document, 0o600); err != nil {
		fail("write %s: %v", *out, err)
	}
	fmt.Fprintf(os.Stderr, "signed %d apps into %s\n", len(catalog), *out)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "catalog-sign: "+format+"\n", args...)
	os.Exit(1)
}
