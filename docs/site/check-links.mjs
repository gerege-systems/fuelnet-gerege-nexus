/**
 * Fails the build if the generated site links to something that is not there.
 *
 *   node check-links.mjs        (run after build.mjs)
 *
 * Only internal links are checked. Whether an external URL is alive is not
 * something this repository can hold true, and a link checker that reaches the
 * network turns an unrelated outage into a failed documentation build.
 *
 * Three defects are caught, all of which look fine in the Markdown source:
 *
 *   1. A link to a page that was never published — usually a document added to
 *      docs/ and not to the PAGES list in build.mjs.
 *   2. A `.md` href that survived rewriting, which a browser downloads instead
 *      of rendering.
 *   3. A `#fragment` naming a heading that no longer exists, which lands the
 *      reader silently at the top of the page.
 */
import {existsSync, readFileSync, readdirSync} from "node:fs";
import {dirname, join} from "node:path";
import {fileURLToPath} from "node:url";

const DIST = join(dirname(fileURLToPath(import.meta.url)), "dist");

if (!existsSync(DIST)) {
  console.error("dist/ is missing — run `npm run build` first.");
  process.exit(1);
}

const pages = readdirSync(DIST).filter((f) => f.endsWith(".html"));
const idsByPage = new Map(
  pages.map((f) => [
    f,
    new Set([...readFileSync(join(DIST, f), "utf8").matchAll(/id="([^"]+)"/g)].map((m) => m[1])),
  ]),
);

const problems = [];

for (const page of pages) {
  const html = readFileSync(join(DIST, page), "utf8");
  for (const [, href] of html.matchAll(/(?:href|src)="([^"]+)"/g)) {
    if (/^(https?:|mailto:|data:|#)/.test(href)) continue;

    const [path, fragment] = href.split("#");
    if (!path) continue;

    if (/\.md$/.test(path)) {
      problems.push(`${page} → ${href} (Markdown link was not rewritten)`);
      continue;
    }
    if (!existsSync(join(DIST, path))) {
      problems.push(`${page} → ${href} (no such file in dist)`);
      continue;
    }
    if (fragment && idsByPage.has(path) && !idsByPage.get(path).has(decodeURIComponent(fragment))) {
      problems.push(`${page} → ${href} (no heading with that id)`);
    }
  }
}

if (problems.length) {
  console.error(`${problems.length} broken link(s):\n`);
  for (const p of problems) console.error(`  ${p}`);
  process.exit(1);
}

console.log(`${pages.length} pages, no broken internal links.`);
