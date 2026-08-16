# Deployment copy

This directory is mounted into the frontend container at `/brand`, read-only.

A deployment that says something different from the rest — a different
positioning, not a different name — puts a `copy.json` here and points
`BRAND_COPY_FILE=/brand/copy.json` at it:

```json
{
  "website.view.hero_lede": { "mn": "…", "en": "…" },
  "website.message.footer_note": { "en": "…" }
}
```

Keys are the ones in `frontend/lib/i18n/`. Overrides are matched per key and
per language, so writing English overrides English and leaves every other
language reading as it did. `{brand}` still works inside an override.

Read once at startup: a change takes effect when the container restarts, the
same moment `BRAND_NAME` would.

Empty is the normal state. Most deployments differ by name and logo alone, and
those are environment variables — see `.env.example`.
