import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");
const slides = html.match(/<section class="slide(?: [^"]*)?">/g) || [];
const notes = html.match(/<aside class="notes">\[Sources\]/g) || [];
const pages = [...html.matchAll(/<span class="pg">(\d{2})<\/span>/g)].map(
  (match) => match[1],
);
const scripts = [...html.matchAll(/<script>([\s\S]*?)<\/script>/g)].map(
  (match) => match[1],
);

assert.equal(slides.length, 20, "the public deck must contain exactly 20 slides");
assert.equal(notes.length, slides.length, "every slide must have a [Sources] note");
assert.deepEqual(
  pages,
  Array.from({ length: 20 }, (_, index) => String(index + 1).padStart(2, "0")),
  "visible slide numbers must be sequential",
);
assert.match(html, /<title>FuelNet/);
assert.doesNotMatch(html, /<script[^>]+src=/, "the deck must remain standalone");
assert.doesNotMatch(html, /<link[^>]+stylesheet/, "the deck must remain standalone");
assert.doesNotMatch(html, /Gerege Nexus — Танилцуулга/);
assert.equal(scripts.length, 1, "the deck must have one inline controller");
assert.doesNotThrow(() => new Function(scripts[0]), "the deck controller must parse");
assert.match(
  html,
  /stage\.style\.transform='translate\(-50%,-50%\) scale\('/,
  "the scaled slide must remain centered on narrow mobile viewports",
);
assert.match(html, /height:calc\(100dvh - var\(--controls-space\)\)/);
assert.match(html, /visualViewport\.addEventListener\('resize',fit\)/);
assert.equal(
  (html.match(/<section\b/g) || []).length,
  (html.match(/<\/section>/g) || []).length,
  "all slide sections must be closed",
);

for (const diagram of [
  "chain-map",
  "loop-diagram",
  "score",
  "adapter",
  "edge-diagram",
  "topology",
  "delta",
  "hash-chain",
  "roadmap",
]) {
  assert.match(html, new RegExp(`class="[^"]*${diagram}`), `${diagram} diagram is missing`);
}

console.log(
  `FuelNet deck: ${slides.length} slides, ${notes.length} source notes, 9 core diagrams, standalone HTML.`,
);
