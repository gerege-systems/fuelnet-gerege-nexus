/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// Vector tiles, served from this origin. See lib/fuel/tileProxy.ts for why the
// map does not fetch them from the tile host directly.

import { proxyMapAsset } from "@/lib/fuel/tileProxy";

export async function GET(_request: Request, context: { params: Promise<{ tile: string[] }> }) {
  const { tile } = await context.params;
  // z/x/y and nothing else. A path segment that is not a number is not a tile
  // request, and refusing it here keeps this route from becoming a general
  // purpose fetcher pointed at whatever an attacker appends.
  if (tile.length !== 3 || !tile.every((part) => /^\d+$/.test(part))) {
    return new Response("expected /tiles/{z}/{x}/{y}", { status: 400 });
  }
  return proxyMapAsset(`tiles/${tile.join("/")}`);
}
