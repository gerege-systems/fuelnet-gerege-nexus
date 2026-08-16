/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * catalog-diff compares two registries' catalogues byte for byte.
 */
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// The gate the cutover has to pass.
//
// A new App Store instance is brought up beside the old one, given the same
// data, and asked the same question. If the two answer with different bytes,
// the cutover does not happen — because the signature covers the raw bytes of
// the apps array, and a client that rejects the document does so silently,
// carrying on with the catalogue it already has while nothing looks wrong
// anywhere.
//
// Comparing decoded JSON would miss exactly the failures worth catching: a
// field order that changed, an escape that is now written differently, a number
// formatted with a trailing zero. Those compare equal as values and are a
// different document to a verifier.
//
// The signature and generated_at are excluded from the comparison and checked
// separately. Two registries signing with different keys produce different
// signatures over identical content, which is expected during a parallel run;
// what must be identical is what was signed.
//
// Usage:
//
//	catalog-diff -old https://appstore.gerege.mn -new https://appstore-next.gerege.mn
//	catalog-diff -old ... -new ... -platform 1.0.0 -channel stable
func main() {
	var (
		oldURL   = flag.String("old", "", "the registry serving today")
		newURL   = flag.String("new", "", "the registry that would replace it")
		platform = flag.String("platform", "1.0.0", "the platform version to ask as")
		channel  = flag.String("channel", "stable", "stable or beta")
		timeout  = flag.Duration("timeout", 30*time.Second, "per-request timeout")
	)
	flag.Parse()
	if *oldURL == "" || *newURL == "" {
		fail("both -old and -new are required")
	}

	client := &http.Client{Timeout: *timeout}
	before, err := fetch(client, *oldURL, *platform, *channel)
	if err != nil {
		fail("old registry: %v", err)
	}
	after, err := fetch(client, *newURL, *platform, *channel)
	if err != nil {
		fail("new registry: %v", err)
	}

	oldDoc, err := parse(before)
	if err != nil {
		fail("old registry: %v", err)
	}
	newDoc, err := parse(after)
	if err != nil {
		fail("new registry: %v", err)
	}

	// The bytes that were signed, compared as bytes.
	if !bytes.Equal(oldDoc.Apps, newDoc.Apps) {
		fmt.Fprintf(os.Stderr, "the two registries do not produce the same catalogue.\n\n")
		describe(oldDoc.Apps, newDoc.Apps)
		os.Exit(1)
	}

	fmt.Printf("identical: %d bytes over %s/%s\n", len(oldDoc.Apps), *channel, *platform)
	if oldDoc.KeyID != newDoc.KeyID {
		// Expected during a parallel run, and worth saying out loud: every
		// instance in the field pins a key, and they must be given the new one
		// before the upstream moves.
		fmt.Printf("note: signed by different keys (%q → %q). Every instance's "+
			"APPSTORE_PUBLIC_KEY must carry the new one before the cutover.\n",
			oldDoc.KeyID, newDoc.KeyID)
	}
	fmt.Println("\nThe catalogue is byte-identical. The cutover may proceed.")
}

// document is the part of the catalogue this compares. apps is kept as raw
// JSON deliberately: decoding and re-encoding it would erase the differences
// this exists to find.
type document struct {
	GeneratedAt string          `json:"generated_at"`
	KeyID       string          `json:"key_id"`
	Apps        json.RawMessage `json:"apps"`
	Signature   string          `json:"signature"`
}

func parse(raw []byte) (*document, error) {
	var doc document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("the answer is not a catalogue: %w", err)
	}
	if len(doc.Apps) == 0 {
		return nil, fmt.Errorf("the catalogue carries no apps")
	}
	if doc.Signature == "" {
		return nil, fmt.Errorf("the catalogue is unsigned")
	}
	return &doc, nil
}

func fetch(client *http.Client, base, platform, channel string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/v1/registry/catalog?platform=%s&channel=%s",
		strings.TrimSuffix(base, "/"), platform, channel)
	res, err := client.Get(url) //nolint:noctx // a one-shot CLI with a client timeout
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("answered %d", res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, 32<<20))
}

// describe finds the first byte that differs and shows its neighbourhood,
// because "the documents differ" is not something anybody can act on.
func describe(before, after []byte) {
	limit := min(len(before), len(after))
	at := limit
	for i := range limit {
		if before[i] != after[i] {
			at = i
			break
		}
	}
	if at == limit && len(before) != len(after) {
		fmt.Fprintf(os.Stderr, "identical for %d bytes, then the lengths differ: old %d, new %d\n",
			limit, len(before), len(after))
	} else {
		fmt.Fprintf(os.Stderr, "first difference at byte %d\n", at)
	}
	fmt.Fprintf(os.Stderr, "\n  old: %s\n  new: %s\n", window(before, at), window(after, at))
}

func window(data []byte, at int) string {
	const around = 60
	start, end := max(0, at-around), min(len(data), at+around)
	return fmt.Sprintf("…%s…", data[start:end])
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "catalog-diff: "+format+"\n", args...)
	os.Exit(1)
}
