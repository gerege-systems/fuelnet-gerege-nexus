import fs from "node:fs";

import type { Locale } from "./i18n";

/**
 * Copy one deployment says differently from every other.
 *
 * `BRAND_*` already carries a deployment's identity — the name, the logo, the
 * colour — and `{brand}` puts the name inside translations without anybody
 * editing them. That covers a rename. It does not cover a *repositioning*: a
 * health platform's landing page does not describe itself the way the general
 * platform does, and the difference is a paragraph, in seven languages, not a
 * noun.
 *
 * Until now that was a code change, which is to say a fork. Gerege Salus was
 * forked from this repository for about eighty lines of translated prose, and
 * carried a full copy of the platform to hold them; every security fix then had
 * to be applied twice. This is the eighty lines given a home in configuration,
 * so the fork does not have to exist.
 *
 * What it is not: a translation system. Overrides are read once at startup from
 * a file the operator mounts, they are matched per locale, and a key nobody
 * overrode reads exactly as it did before.
 */
export type BrandCopy = Record<string, Partial<Record<Locale, string>>>;

/**
 * Read once per process.
 *
 * A file is not an environment variable — it can be edited under a running
 * container — but treating it as one keeps this predictable: a deployment's
 * wording changes when the deployment restarts, the same moment `BRAND_NAME`
 * would take effect, rather than at whichever request happened to reread it.
 */
let cached: BrandCopy | undefined;

/**
 * Overrides from the environment.
 *
 *   BRAND_COPY_FILE  path to a JSON file, which is how a deployment supplies
 *                    paragraphs — mounted beside the logo it already mounts.
 *   BRAND_COPY       the same JSON inline, for the deployment that overrides a
 *                    line or two and would rather not carry a file for it.
 *
 * The file wins when both are set: a mounted file is the more deliberate of the
 * two, and silently preferring the string in the compose file would be a
 * surprise at exactly the wrong moment.
 */
export function brandCopyFromEnv(env: NodeJS.ProcessEnv = process.env): BrandCopy {
  if (cached) return cached;
  cached = parse(read(env));
  return cached;
}

function read(env: NodeJS.ProcessEnv): string {
  const path = (env.BRAND_COPY_FILE ?? "").trim();
  if (path) {
    try {
      return fs.readFileSync(path, "utf8");
    } catch (error) {
      // A named file that cannot be read is an operator's mistake, not a
      // reason to refuse to serve: the shell comes up in this product's own
      // words, which is the state it was in before anybody named a file.
      console.warn(`BRAND_COPY_FILE could not be read (${path}):`, error);
      return "";
    }
  }
  return (env.BRAND_COPY ?? "").trim();
}

/**
 * Everything that is not a locale-to-string entry is dropped, and dropped
 * loudly.
 *
 * The values reach the screen, so the shape is checked rather than trusted —
 * the same bargain `brandEnv` makes with a logo address. There is no injection
 * to fear (React escapes what it renders and these are rendered as text), but a
 * malformed override that half-applied would be found by a reader rather than
 * by whoever wrote it.
 */
function parse(raw: string): BrandCopy {
  if (!raw) return {};

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch (error) {
    console.warn("brand copy is not valid JSON; deployment wording is ignored:", error);
    return {};
  }
  if (!isRecord(parsed)) {
    console.warn("brand copy must be a JSON object of translation keys; deployment wording is ignored");
    return {};
  }

  const copy: BrandCopy = {};
  for (const [key, value] of Object.entries(parsed)) {
    if (!isRecord(value)) {
      console.warn(`brand copy: "${key}" is not an object of languages; ignored`);
      continue;
    }
    const entry: Partial<Record<Locale, string>> = {};
    for (const [locale, text] of Object.entries(value)) {
      if (typeof text !== "string" || !text.trim()) {
        console.warn(`brand copy: "${key}.${locale}" is not text; ignored`);
        continue;
      }
      entry[locale as Locale] = text;
    }
    if (Object.keys(entry).length > 0) copy[key] = entry;
  }
  return copy;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
