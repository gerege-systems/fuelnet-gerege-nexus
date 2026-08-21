#!/usr/bin/env node
/**
 * Every link the shell offers has to lead somewhere.
 *
 *   npm run links:check
 *
 * Two files hand a person a list of places to go before they have navigated
 * anywhere: `app/line/[line]/lines.ts`, which is the home screen each native
 * shell draws, and `app/manifest.ts`, whose shortcuts appear in the installed
 * app's launcher. Both are plain data, so nothing checks them — a route can be
 * deleted and the tile stays, and the failure is a person tapping something and
 * getting a blank page.
 *
 * That happened. When the departed apps' screens were removed on 2026-08-21,
 * fourteen of the twenty-six line tiles and two of the three launcher shortcuts
 * were left pointing at pages that no longer existed. The build was green: a
 * string is a string.
 *
 * So the strings are resolved against the App Router's own file layout. A href
 * must have a page.tsx at the path it names, or a dynamic segment that could
 * match it.
 *
 * Exits non-zero on any link that leads nowhere.
 */
import { readFile, readdir, stat } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const APP = path.join(HERE, "..", "app");

const exists = async (p) => stat(p).then(() => true, () => false);

/** Every route this application serves, as a set of "/a/b" strings. */
async function routes(dir = APP, prefix = "") {
  const found = new Set();
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    // Route groups (auth) and private folders (_lib) do not appear in the URL.
    if (entry.name.startsWith("_")) continue;
    const segment = entry.name.startsWith("(") ? "" : `/${entry.name}`;
    const child = path.join(dir, entry.name);
    if (await exists(path.join(child, "page.tsx"))) found.add(prefix + segment || "/");
    for (const nested of await routes(child, prefix + segment)) found.add(nested);
  }
  if (prefix === "" && (await exists(path.join(APP, "page.tsx")))) found.add("/");
  return found;
}

/** Does `href` match a served route, allowing for [dynamic] segments? */
function served(href, all) {
  if (all.has(href)) return true;
  const parts = href.split("/");
  return [...all].some((route) => {
    const candidate = route.split("/");
    if (candidate.length !== parts.length) return false;
    return candidate.every((seg, i) => seg === parts[i] || /^\[.*\]$/.test(seg));
  });
}

const all = await routes();

const sources = [
  { file: path.join(APP, "line", "[line]", "lines.ts"), pattern: /href: "([^"]+)"/g },
  { file: path.join(APP, "manifest.ts"), pattern: /url: "([^"]+)"/g },
];

let broken = 0;
let checked = 0;
for (const { file, pattern } of sources) {
  const source = await readFile(file, "utf8");
  const rel = path.relative(path.join(HERE, ".."), file);
  for (const match of source.matchAll(pattern)) {
    const href = match[1];
    // Only in-app paths. An external_url belongs to somebody else.
    if (!href.startsWith("/") || href.startsWith("//")) continue;
    checked++;
    if (!served(href.split("?")[0], all)) {
      broken++;
      console.error(`${rel}: ${href} — no page.tsx serves this path`);
    }
  }
}

if (broken > 0) {
  console.error(`
${broken} of ${checked} shell links lead nowhere.

These are the tiles a native shell draws on its home screen and the shortcuts in
the installed app's launcher — the first things a person sees, and the only
links in this application that no router check covers. Point them at a route
that exists, or remove them.`);
  process.exit(1);
}

console.log(`${checked} shell links, all served by a real page.`);
