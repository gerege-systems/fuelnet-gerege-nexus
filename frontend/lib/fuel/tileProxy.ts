/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

/**
 * Serving map tiles from our own origin.
 *
 * # Why a proxy and not a direct fetch
 *
 * The tiles live on another product's host today. Its nginx and the Martin
 * behind it both add `Access-Control-Allow-Origin`, so a browser sees the
 * header twice — once echoing our origin, once `*` — and refuses the response:
 * "contains multiple values ... but only one is allowed". Every tile fails and
 * the map draws its background colour and nothing else.
 *
 * That is somebody else's nginx to fix, and waiting on it would be the smaller
 * problem anyway. A deployment reaching across the internet to another team's
 * tile server on every pan is resting on their uptime, their bandwidth and
 * their CORS configuration without appearing anywhere in their capacity
 * planning. Same-origin removes all three at once: there is no CORS on a
 * same-origin request, so no header to get wrong.
 *
 * # This is the seam production wants
 *
 * `MAP_TILES_UPSTREAM` is where the bytes actually come from. Point it at a
 * Martin of our own over a copy of the .mbtiles and nothing in the client
 * changes — the style already asks for `/tiles/{z}/{x}/{y}` on this origin.
 */

const DEFAULT_UPSTREAM = "https://monzasvar.mn";

/** Where the tiles and glyphs are fetched from, server-side. */
export function upstream(): string {
  return (process.env.MAP_TILES_UPSTREAM || DEFAULT_UPSTREAM).replace(/\/$/, "");
}

/**
 * Fetch one asset from the upstream and hand it back unchanged.
 *
 * A 204 is passed through as a 204 rather than turned into an error: Martin
 * answers it for a tile with no data in it, which is most of the country
 * outside Ulaanbaatar, and MapLibre reads it as "nothing here" and moves on.
 * Turning it into a 404 would make an empty tile look like a broken one.
 */
export async function proxyMapAsset(path: string): Promise<Response> {
  let upstreamResponse: Response;
  try {
    upstreamResponse = await fetch(`${upstream()}/${path}`, {
      // The upstream's caching is what we want to inherit; ours is set below.
      cache: "no-store",
      headers: { Accept: "*/*" },
    });
  } catch {
    // The tile host is unreachable. 502 rather than 500: this process is fine,
    // the thing it depends on is not, and a monitor should be able to tell.
    return new Response("map upstream unreachable", { status: 502 });
  }

  if (upstreamResponse.status === 204) {
    return new Response(null, { status: 204, headers: cacheHeaders() });
  }
  if (!upstreamResponse.ok) {
    return new Response(null, { status: upstreamResponse.status });
  }

  const headers = cacheHeaders();
  const type = upstreamResponse.headers.get("content-type");
  if (type) headers.set("content-type", type);

  // `content-encoding` is deliberately NOT forwarded. The upstream serves these
  // gzipped, and fetch decompresses transparently — so by the time the body is
  // in hand it is plain protobuf. Passing the header on told the client to
  // gunzip bytes that had already been gunzipped, and every tile failed to
  // decode. The runtime re-compresses on the way out if the client asked for it.
  return new Response(upstreamResponse.body, { status: 200, headers });
}

/**
 * A day, and stale-while-revalidate for a week.
 *
 * Tiles change when somebody rebuilds the .mbtiles, which is a deliberate act
 * measured in months. The cost of a stale tile is a building drawn where it was
 * last week; the cost of no caching is every pan going back over the network.
 */
function cacheHeaders(): Headers {
  return new Headers({
    "cache-control": "public, max-age=86400, stale-while-revalidate=604800",
  });
}
