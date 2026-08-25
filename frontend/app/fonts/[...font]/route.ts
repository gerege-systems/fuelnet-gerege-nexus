/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// Glyph ranges for map labels, served from this origin. Same reasoning as the
// tiles beside it — see lib/fuel/tileProxy.ts.

import { proxyMapAsset } from "@/lib/fuel/tileProxy";

export async function GET(_request: Request, context: { params: Promise<{ font: string[] }> }) {
  const { font } = await context.params;
  // {fontstack}/{range}.pbf — a font name and a range like 0-255.pbf.
  if (font.length !== 2 || !/^\d+-\d+\.pbf$/.test(font[1])) {
    return new Response("expected /fonts/{fontstack}/{range}.pbf", { status: 400 });
  }
  return proxyMapAsset(`fonts/${encodeURIComponent(font[0])}/${font[1]}`);
}
