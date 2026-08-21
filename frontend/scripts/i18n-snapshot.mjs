#!/usr/bin/env node
/**
 * Prints what `t()` would return for every key in every language.
 *
 *   node scripts/i18n-snapshot.mjs > before.json
 *   git stash && node scripts/i18n-snapshot.mjs > after.json && git stash pop
 *   diff before.json after.json
 *
 * Written for one job: proving that moving the apps' dictionaries out of
 * lib/i18n/index.tsx and into a runtime registry changed no string anybody
 * reads. A refactor of a translation layer is exactly the kind that looks
 * finished because it compiles — every failure mode is a screen quietly
 * rendering English, in a language nobody on the team reads, on a route nobody
 * opened during review.
 *
 * It reads both arrangements, so it can be run on either side of the change:
 * the old one, where lib/i18n/locales/<locale>.ts holds every key, and the new
 * one, where each locale is a directory of core.ts plus one file per app. The
 * resolution order it applies is the one in index.tsx, minus the deployment's
 * own brand copy — that is configuration, not translation.
 *
 * Kept in the tree afterwards rather than deleted: the next person to move a
 * dictionary needs the same proof, and writing it again is how it gets written
 * slightly differently.
 */
import { readFile, readdir, stat } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const I18N = path.join(HERE, "..", "lib", "i18n");
const LOCALES = ["mn", "ar", "zh", "en", "fr", "ru", "es"];
const OVERLAY_LOCALES = ["ar", "zh", "fr", "ru", "es"];

/**
 * Reads a dictionary module without a TypeScript loader — the same trick
 * scripts/i18n-check.mjs uses. These files are `export const x = {...}` where
 * every value is a literal: data, not code.
 */
async function load(file) {
  const src = await readFile(file, "utf8");
  const start = src.indexOf("{", src.indexOf("export const"));
  const end = src.lastIndexOf("}");
  if (start === -1 || end === -1) throw new Error(`cannot read an object literal from ${file}`);
  return Function(`"use strict"; return (${src.slice(start, end + 1)});`)();
}

const exists = async (p) => stat(p).then(() => true, () => false);

// The hand-authored source: mn and en, one file per module. Unchanged by the
// split, and read the same way on both sides.
const dictionary = {};
for (const f of [path.join(I18N, "base.ts"), path.join(I18N, "web.ts")]) {
  Object.assign(dictionary, await load(f));
}
for (const f of (await readdir(path.join(I18N, "addons"))).sort()) {
  if (f.endsWith(".ts")) Object.assign(dictionary, await load(path.join(I18N, "addons", f)));
}

// The generated overlays, from whichever layout is on disk.
const overlays = {};
for (const locale of OVERLAY_LOCALES) {
  const flat = path.join(I18N, "locales", `${locale}.ts`);
  const dir = path.join(I18N, "locales", locale);
  const merged = {};
  if (await exists(flat)) {
    Object.assign(merged, await load(flat));
  } else {
    for (const f of (await readdir(dir)).sort()) {
      if (f.endsWith(".ts")) Object.assign(merged, await load(path.join(dir, f)));
    }
  }
  overlays[locale] = merged;
}

const snapshot = {};
for (const key of Object.keys(dictionary).sort()) {
  const entry = dictionary[key];
  snapshot[key] = Object.fromEntries(
    LOCALES.map((locale) => [locale, overlays[locale]?.[key] || entry?.[locale] || entry?.en || key]),
  );
}

process.stdout.write(`${JSON.stringify(snapshot, null, 1)}\n`);
