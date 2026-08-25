/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

/**
 * Put MapLibre's worker somewhere the browser can actually fetch it.
 *
 * MapLibre parses vector tiles on a worker thread, and from v5 that worker is
 * an ES module: `new Worker(url, { type: "module" })`, with the url derived
 * from `import.meta.url` of the bundled library. Under Turbopack that url
 * resolves to a path the dev server answers with HTML, and the browser refuses
 * it — "Failed to load module script: the server responded with a
 * non-JavaScript MIME type of text/html". The map then renders nothing at all,
 * with no error on the map object, because the failure is in the worker's
 * loader rather than in anything MapLibre awaits.
 *
 * So the worker is served as a static asset instead and named explicitly with
 * `setWorkerUrl`. Two files, because the worker imports the shared chunk by a
 * relative path — they have to land in the same directory.
 *
 * Copied at build time rather than committed: a copy checked into the tree is
 * a copy that silently disagrees with package.json the first time anybody
 * upgrades maplibre-gl. `public/maplibre/` is gitignored for the same reason.
 */

import { copyFile, mkdir } from "node:fs/promises";
import { createRequire } from "node:module";
import { dirname, join } from "node:path";

const require = createRequire(import.meta.url);
const dist = dirname(require.resolve("maplibre-gl/dist/maplibre-gl.mjs"));
const target = join(process.cwd(), "public", "maplibre");

// The worker and the chunk it imports. Keep them together: the import inside
// the worker is relative.
const FILES = ["maplibre-gl-worker.mjs", "maplibre-gl-shared.mjs"];

await mkdir(target, { recursive: true });
for (const name of FILES) {
  await copyFile(join(dist, name), join(target, name));
}
console.log(`maplibre: copied ${FILES.length} files to public/maplibre/`);
