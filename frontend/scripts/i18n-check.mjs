#!/usr/bin/env node
/**
 * Audits the dictionary against the language overlays.
 *
 *   npm run i18n:check          # exits non-zero on any gap
 *   npm run i18n:check -- --warn # reports and exits zero
 *
 * Three failures this catches, all of which are silent at runtime:
 *
 *   1. An **orphaned** overlay key — a translation whose dictionary key was
 *      renamed or deleted. Nothing reads it, so the screen it was written for
 *      quietly renders English forever.
 *   2. An **untranslated** key — one the dictionary has and the overlay does
 *      not. `t()` falls back to English by design, which is the right runtime
 *      behaviour and the wrong thing to discover from a screenshot. A screen
 *      showing Mongolian menus, Chinese headings and English buttons at the
 *      same time is what a pile of these looks like from the outside.
 *   3. A **misfiled** translation — an app's key sitting in another file. The
 *      overlays are one file per app now so that an app's words travel with the
 *      app; a `documents.*` line written into core.ts still renders, and quietly
 *      undoes the split.
 *
 * `--warn` exists because the second class is not empty and was never going to
 * be fixed by the commit that put this in CI: 257 keys per language are
 * untranslated today. Running with a known backlog and printing it is how the
 * number is watched; running not at all is how it grew. The other two classes
 * are always fatal — an orphan and a misfiling are mistakes, not backlog.
 */

import path from "node:path";
import { fileURLToPath } from "node:url";

import { OVERLAY_LOCALES, loadDictionary, loadOverlay, loadModule, overlayFile } from "./i18n-layout.mjs";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const I18N = path.join(HERE, "..", "lib", "i18n");
const WARN_ONLY = process.argv.includes("--warn");

const { dictionary, bucketOf } = await loadDictionary(I18N);
const keys = Object.keys(dictionary);

let fatal = false;
let backlog = 0;

// mn and en are hand-authored on the entry itself, so they are checked here
// rather than in an overlay.
for (const [key, entry] of Object.entries(dictionary)) {
  for (const required of ["mn", "en"]) {
    if (!entry?.[required]) {
      console.error(`✗ ${key}: missing ${required} in the source dictionary`);
      fatal = true;
    }
  }
}

for (const locale of OVERLAY_LOCALES) {
  const overlay = await loadOverlay(I18N, locale);
  const orphaned = Object.keys(overlay).filter((k) => !(k in dictionary));
  const untranslated = keys.filter((k) => !overlay[k]);

  // And the third class: every key has to be in the file its app owns.
  const misfiled = [];
  for (const bucket of new Set(Object.values(bucketOf))) {
    let entries;
    try {
      entries = await loadModule(overlayFile(I18N, locale, bucket));
    } catch {
      continue; // No translations for that app in this language yet.
    }
    for (const key of Object.keys(entries)) {
      if (bucketOf[key] && bucketOf[key] !== bucket) {
        misfiled.push(`${key} is in ${bucket}.ts but belongs to ${bucketOf[key]}.ts`);
      }
    }
  }

  backlog += untranslated.length;
  if (orphaned.length || misfiled.length) fatal = true;

  const clean = !orphaned.length && !untranslated.length && !misfiled.length;
  console.log(
    `${clean ? "✓" : "✗"} ${locale}: ${keys.length - untranslated.length}/${keys.length} translated` +
      (untranslated.length ? `, ${untranslated.length} untranslated` : "") +
      (orphaned.length ? `, ${orphaned.length} orphaned` : "") +
      (misfiled.length ? `, ${misfiled.length} misfiled` : ""),
  );
  for (const k of orphaned) console.error(`    orphaned: ${k}`);
  for (const m of misfiled) console.error(`    misfiled: ${m}`);
  if (!WARN_ONLY) for (const k of untranslated) console.error(`    untranslated: ${k}`);
}

console.log(`\n${keys.length} keys, ${OVERLAY_LOCALES.length} overlays, ${backlog} untranslated in total.`);

if (fatal || (backlog && !WARN_ONLY)) {
  console.error("Run `npm run i18n:translate -- --locale <code> --write` to fill the gaps.");
  process.exit(1);
}
