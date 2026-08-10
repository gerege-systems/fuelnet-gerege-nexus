/*
 * Gerege Nexus — App Store registry
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

package appstore

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appcatalog"
)

// SeedFromCatalogFile imports catalog/apps.json and its manifests as the first
// publisher's published apps.
//
// This is the one-time bridge from "the catalogue is a file in the platform
// repository" to "the catalogue is what the registry publishes". It is
// idempotent and only ever adds: an app that is already here keeps its
// publisher and its versions, so running it again after somebody has published
// through the console cannot undo their work.
//
// It runs at startup when the registry is empty rather than as a separate
// operational step, because a registry with no apps is not a state worth
// deploying and then remembering to fix.
func (s *Store) SeedFromCatalogFile(ctx context.Context, path, publisherSlug, publisherName, ownerSub string) error {
	catalog, err := appcatalog.LoadFile(path, "")
	if err != nil {
		return fmt.Errorf("read the bundled catalog: %w", err)
	}

	publisher, err := s.PublisherByOwner(ctx, ownerSub)
	if errors.Is(err, ErrNotFound) {
		publisher, err = s.CreatePublisher(ctx, &Publisher{
			Slug: publisherSlug, Name: publisherName, OwnerSub: ownerSub,
		})
		if err != nil {
			return fmt.Errorf("create the seed publisher: %w", err)
		}
		// The platform's own apps are published by the platform's own team, so
		// this one publisher is verified by construction. Every other publisher
		// is verified by a person.
		if err := s.SetPublisherVerified(ctx, publisher.ID, true); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	for _, entry := range catalog {
		app := &App{
			ID: entry.ID, PublisherID: publisher.ID, Slug: entry.Slug,
			Type: entry.Manifest.Type, Name: entry.Name, Description: entry.Description,
			IconURL: entry.IconURL, Category: entry.Category, Visibility: entry.Visibility,
		}
		if app.Type == "" {
			app.Type = appcatalog.TypeModule
		}

		existing, err := s.AppByID(ctx, entry.ID)
		switch {
		case errors.Is(err, ErrNotFound):
			if err := s.UpsertApp(ctx, app, entry.Translations); err != nil {
				return fmt.Errorf("seed app %s: %w", entry.ID, err)
			}
		case err != nil:
			return err
		case existing.PublisherID != publisher.ID:
			// Somebody has taken this app over through the console. The seed is
			// a starting point, not an authority.
			slog.Info("seed: app already belongs to another publisher; leaving it alone", "app_id", entry.ID)
			continue
		default:
			if err := s.UpsertApp(ctx, app, entry.Translations); err != nil {
				return fmt.Errorf("seed app %s: %w", entry.ID, err)
			}
		}

		minPlatform := entry.Manifest.Platform
		if minPlatform == "" {
			minPlatform = ">=0.1.0"
		}
		version := &Version{
			AppID: entry.ID, Version: entry.Version, Channel: "stable",
			MinPlatform: minPlatform, Manifest: entry.Manifest,
			Status: StatusPublished, SubmittedBy: "seed",
		}

		var external *ExternalRegistration
		if entry.Manifest.IsExternal() && entry.Manifest.External != nil {
			external = &ExternalRegistration{
				AppID: entry.ID, LaunchURL: entry.Manifest.External.LaunchURL,
				SSOClientID: entry.Manifest.External.SSOClientID,
				Scopes:      entry.Manifest.External.Scopes,
				Embed:       entry.Manifest.External.Embed,
				HealthURL:   entry.Manifest.External.HealthURL,
			}
		}

		saved, err := s.SubmitVersion(ctx, version, external)
		if errors.Is(err, ErrConflict) {
			// Already imported. A published version is immutable, so there is
			// nothing to update and nothing to worry about.
			continue
		}
		if err != nil {
			return fmt.Errorf("seed version %s of %s: %w", entry.Version, entry.ID, err)
		}
		if err := s.DecideVersion(ctx, saved.ID, "publish", "seed", "imported from the bundled catalog"); err != nil {
			return fmt.Errorf("publish seeded version %s of %s: %w", entry.Version, entry.ID, err)
		}
		slog.Info("seeded a catalog app", "app_id", entry.ID, "version", entry.Version)
	}

	return nil
}

// IsEmpty reports whether anything has been published yet.
func (s *Store) IsEmpty(ctx context.Context) (bool, error) {
	var count int
	err := s.db.QueryRow(ctx, `SELECT count(*) FROM store_apps`).Scan(&count)
	return count == 0, err
}
