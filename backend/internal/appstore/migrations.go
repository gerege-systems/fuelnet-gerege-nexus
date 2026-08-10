package appstore

import "embed"

// Migrations is the registry's schema, carried inside the binary.
//
// It is one service with one schema deployed as one image, so a separate
// migration container — the shape the platform uses, where the API and the
// migrator are the same image run two ways — would be a second thing to keep in
// step for no benefit here.
//
//go:embed all:migrations
var Migrations embed.FS
