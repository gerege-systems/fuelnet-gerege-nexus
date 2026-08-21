/**
 * Where the dictionary lives on disk, for the scripts that read and write it.
 *
 * Three scripts used to each open `lib/i18n/locales/<locale>.ts` by name. The
 * overlays are one directory per language now, holding `core.ts` and one file
 * per app, so "the overlay for ru" is no longer a path — it is a merge, and
 * writing a new translation means knowing which app owns the key. That answer
 * is derived here, once, from the addon files themselves rather than from a
 * list somebody has to keep in step.
 */
import { readFile, readdir } from "node:fs/promises";
import path from "node:path";

/**
 * The platform's own dictionaries. Everything else under addons/ belongs to an
 * app and travels with it — see lib/i18n/registry.ts.
 *
 * `base` and `web` are core too; they sit above addons/ and are added below.
 */
export const CORE_ADDONS = [
  "access",
  "appearance",
  "app_store",
  "auth",
  "core",
  "cp",
  "emailverify",
  "integrations",
  "modules",
  "sharing",
  "website",
];

export const OVERLAY_LOCALES = ["ar", "zh", "fr", "ru", "es"];

/**
 * Reads a dictionary module without a TypeScript loader. These files are
 * `export const x = {...}` where every value is a literal: data, not code.
 */
export async function loadModule(file) {
  const src = await readFile(file, "utf8");
  const start = src.indexOf("{", src.indexOf("export const"));
  const end = src.lastIndexOf("}");
  if (start === -1 || end === -1) throw new Error(`cannot read an object literal from ${file}`);
  return Function(`"use strict"; return (${src.slice(start, end + 1)});`)();
}

/**
 * The hand-authored mn/en dictionary, plus which bucket each key belongs to.
 *
 * A bucket is "core" or an app's name, and it is also the overlay filename: the
 * ru translation of a `documents.*` key lives in locales/ru/documents.ts.
 */
export async function loadDictionary(i18n) {
  const dictionary = {};
  const bucketOf = {};

  const add = async (file, bucket) => {
    const entries = await loadModule(file);
    for (const key of Object.keys(entries)) bucketOf[key] = bucket;
    Object.assign(dictionary, entries);
  };

  await add(path.join(i18n, "base.ts"), "core");
  await add(path.join(i18n, "web.ts"), "core");
  for (const file of (await readdir(path.join(i18n, "addons"))).sort()) {
    if (!file.endsWith(".ts")) continue;
    const name = file.slice(0, -3);
    await add(path.join(i18n, "addons", file), CORE_ADDONS.includes(name) ? "core" : name);
  }

  return { dictionary, bucketOf };
}

/** Every overlay file for one language, merged the way `t()` sees them. */
export async function loadOverlay(i18n, locale) {
  const dir = path.join(i18n, "locales", locale);
  const merged = {};
  for (const file of (await readdir(dir)).sort()) {
    if (file.endsWith(".ts")) Object.assign(merged, await loadModule(path.join(dir, file)));
  }
  return merged;
}

export const overlayFile = (i18n, locale, bucket) =>
  path.join(i18n, "locales", locale, `${bucket}.ts`);
