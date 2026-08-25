/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

/**
 * The map this deployment draws.
 *
 * MapLibre GL over vector tiles we serve ourselves, rather than Google Maps.
 * The reasoning is not ours and does not need repeating from scratch — the
 * `san` project measured it and wrote it down (DECISIONS.md §6): tiles we host
 * cost the server they run on and nothing per view, carry Mongolian detail no
 * global provider has (khashaa boundaries, local POI names), and can be cached
 * for offline use. A Google Dynamic Map is billed per load, and at the scale
 * this platform is aimed at — every driver in the country, several times a day
 * — per-load pricing is the whole budget.
 *
 * The style below is that project's, ported from Swift. Layer order, colours
 * and zoom breakpoints are unchanged: it is a style somebody has already looked
 * at on a phone in daylight, which is worth more than one that reads well in a
 * diff.
 *
 * # Both URLs are on this origin
 *
 * The tiles themselves come from elsewhere, but the browser never sees that:
 * app/tiles/ and app/fonts/ proxy them, and lib/fuel/tileProxy.ts says why.
 * Briefly — the host serving them today sends `Access-Control-Allow-Origin`
 * twice, which a browser refuses outright, and it belongs to another product
 * anyway. Same-origin has no CORS to get wrong.
 *
 * `MAP_TILES_UPSTREAM` (server-side, no NEXT_PUBLIC_) is where the bytes are
 * fetched from. Point it at a Martin of our own and nothing here changes.
 */

import type { StyleSpecification } from "maplibre-gl";

const DEFAULT_TILES = "/tiles/{z}/{x}/{y}";
const DEFAULT_GLYPHS = "/fonts/{fontstack}/{range}.pbf";

export function tileURL(): string {
  return process.env.NEXT_PUBLIC_MAP_TILES_URL || DEFAULT_TILES;
}

export function glyphsURL(): string {
  return process.env.NEXT_PUBLIC_MAP_GLYPHS_URL || DEFAULT_GLYPHS;
}

/**
 * Mongolia, rounded outward a little so border towns and GPS drift are not
 * pulled back to the capital.
 *
 * Our tiles cover this and nothing else, so a map centred outside it draws the
 * background colour and reads as broken — which is what a traveller, or anybody
 * whose browser guesses the wrong country, would otherwise see.
 */
export const MONGOLIA_BOUNDS = { minLat: 41.4, maxLat: 52.3, minLon: 87.5, maxLon: 120.2 };

/** Sükhbaatar Square — where the map opens when we have nothing better. */
export const ULAANBAATAR = { lat: 47.9185, lon: 106.9175 };

export function isCovered(lat: number, lon: number): boolean {
  if (!Number.isFinite(lat) || !Number.isFinite(lon)) return false;
  if (lat === 0 && lon === 0) return false; // a dropped fix arrives as 0,0
  return (
    lat >= MONGOLIA_BOUNDS.minLat &&
    lat <= MONGOLIA_BOUNDS.maxLat &&
    lon >= MONGOLIA_BOUNDS.minLon &&
    lon <= MONGOLIA_BOUNDS.maxLon
  );
}

export function fuelMapStyle(): StyleSpecification {
  return {
    version: 8,
    name: "Gerege Fuel",
    sources: {
      base: { type: "vector", tiles: [tileURL()], minzoom: 0, maxzoom: 16 },
    },
    glyphs: glyphsURL(),
    layers: [
      { id: "bg", type: "background", paint: { "background-color": "#f5f3ef" } },
      { id: "water", type: "fill", source: "base", "source-layer": "water",
        paint: { "fill-color": "#b6d6f0" } },
      { id: "park", type: "fill", source: "base", "source-layer": "landuse",
        filter: ["in", "class", "park", "grass"],
        paint: { "fill-color": "#d8ecc8", "fill-opacity": 0.7 } },
      { id: "residential", type: "fill", source: "base", "source-layer": "landuse",
        filter: ["==", "class", "residential"],
        paint: { "fill-color": "#eeebe5", "fill-opacity": 0.5 } },
      { id: "khashaa-fill", type: "fill", source: "base", "source-layer": "landuse",
        filter: ["==", "class", "khashaa"], minzoom: 13,
        paint: { "fill-color": "#f5f0e8",
          "fill-opacity": ["interpolate", ["linear"], ["zoom"], 13, 0.1, 15, 0.5] } },
      { id: "khashaa", type: "line", source: "base", "source-layer": "landuse",
        filter: ["==", "class", "khashaa"], minzoom: 14,
        paint: { "line-color": "#c8c4bc", "line-width": 0.8, "line-dasharray": [4, 3] } },
      { id: "building-2d", type: "fill", source: "base", "source-layer": "building",
        minzoom: 13, maxzoom: 15,
        paint: { "fill-color": "#e0dcd6",
          "fill-opacity": ["interpolate", ["linear"], ["zoom"], 13, 0.3, 15, 0.9] } },
      { id: "building-3d", type: "fill-extrusion", source: "base", "source-layer": "building",
        minzoom: 15,
        paint: {
          "fill-extrusion-color": "#ddd8d0",
          "fill-extrusion-height": ["*", ["coalesce", ["get", "render_height"], 3], 5],
          "fill-extrusion-base": ["*", ["coalesce", ["get", "render_min_height"], 0], 5],
          "fill-extrusion-opacity": ["interpolate", ["linear"], ["zoom"], 15, 0.6, 17, 0.9],
        } },
      { id: "building-outline", type: "line", source: "base", "source-layer": "building",
        minzoom: 14, paint: { "line-color": "#ccc7be", "line-width": 0.5 } },

      { id: "road-minor-bg", type: "line", source: "base", "source-layer": "transportation",
        filter: ["in", "class", "minor", "service"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#e0dcd6",
          "line-width": ["interpolate", ["linear"], ["zoom"], 12, 1.5, 16, 7] } },
      { id: "road-minor", type: "line", source: "base", "source-layer": "transportation",
        filter: ["in", "class", "minor", "service"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#ffffff",
          "line-width": ["interpolate", ["linear"], ["zoom"], 12, 0.8, 16, 5] } },
      { id: "road-tertiary-bg", type: "line", source: "base", "source-layer": "transportation",
        filter: ["==", "class", "tertiary"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#d8d3cb",
          "line-width": ["interpolate", ["linear"], ["zoom"], 10, 2, 16, 9] } },
      { id: "road-tertiary", type: "line", source: "base", "source-layer": "transportation",
        filter: ["==", "class", "tertiary"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#ffffff",
          "line-width": ["interpolate", ["linear"], ["zoom"], 10, 1, 16, 7] } },
      { id: "road-secondary-bg", type: "line", source: "base", "source-layer": "transportation",
        filter: ["==", "class", "secondary"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#d4a853",
          "line-width": ["interpolate", ["linear"], ["zoom"], 8, 2, 16, 10] } },
      { id: "road-secondary", type: "line", source: "base", "source-layer": "transportation",
        filter: ["==", "class", "secondary"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#f5deb3",
          "line-width": ["interpolate", ["linear"], ["zoom"], 8, 1, 16, 8] } },
      { id: "road-primary-bg", type: "line", source: "base", "source-layer": "transportation",
        filter: ["==", "class", "primary"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#d4a24a",
          "line-width": ["interpolate", ["linear"], ["zoom"], 6, 2, 16, 12] } },
      { id: "road-primary", type: "line", source: "base", "source-layer": "transportation",
        filter: ["==", "class", "primary"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#f0c77b",
          "line-width": ["interpolate", ["linear"], ["zoom"], 6, 1, 16, 10] } },
      { id: "road-trunk-bg", type: "line", source: "base", "source-layer": "transportation",
        filter: ["in", "class", "trunk", "motorway"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#c9913a",
          "line-width": ["interpolate", ["linear"], ["zoom"], 5, 2.5, 16, 14] } },
      { id: "road-trunk", type: "line", source: "base", "source-layer": "transportation",
        filter: ["in", "class", "trunk", "motorway"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#f0c06b",
          "line-width": ["interpolate", ["linear"], ["zoom"], 5, 1.5, 16, 12] } },
      { id: "rail", type: "line", source: "base", "source-layer": "transportation",
        filter: ["==", "class", "rail"],
        paint: { "line-color": "#bbbbbb", "line-width": 1.5, "line-dasharray": [3, 3] } },

      { id: "road-label", type: "symbol", source: "base", "source-layer": "transportation_name",
        minzoom: 13,
        layout: { "text-field": "{name}", "text-size": 11, "symbol-placement": "line",
          "text-font": ["Noto Sans Regular"] },
        paint: { "text-color": "#777777", "text-halo-color": "#ffffff", "text-halo-width": 1.5 } },

      // Building names, house numbers and points of interest.
      //
      // Dropped when this style was first ported and put back on 2026-08-25:
      // they are most of what makes a Mongolian address findable, and the tiles
      // carry them — a sample z16 tile holds 114 buildings, 40 house numbers
      // and 271 POIs. Their zoom floors are the source style's, so they appear
      // as the map is zoomed into a neighbourhood rather than crowding the city.
      { id: "building-label", type: "symbol", source: "base", "source-layer": "building",
        minzoom: 15, filter: ["has", "name"],
        layout: { "text-field": "{name}", "text-size": 11, "text-font": ["Noto Sans Regular"],
          "text-max-width": 8 },
        paint: { "text-color": "#555555", "text-halo-color": "#ffffff", "text-halo-width": 1.5 } },
      { id: "housenumber", type: "symbol", source: "base", "source-layer": "housenumber",
        minzoom: 16,
        layout: { "text-field": "{housenumber}", "text-size": 10, "text-font": ["Noto Sans Regular"] },
        paint: { "text-color": "#999999", "text-halo-color": "#ffffff", "text-halo-width": 1 } },
      { id: "poi-label", type: "symbol", source: "base", "source-layer": "poi",
        minzoom: 15, filter: ["has", "name"],
        layout: { "text-field": "{name}", "text-size": 11, "text-font": ["Noto Sans Regular"],
          "text-anchor": "center", "text-max-width": 8 },
        paint: { "text-color": "#2a7ab5", "text-halo-color": "#ffffff", "text-halo-width": 1.2 } },

      { id: "place-label", type: "symbol", source: "base", "source-layer": "place",
        filter: ["has", "name"],
        layout: { "text-field": "{name}",
          "text-size": ["match", ["get", "class"], "city", 18, "town", 14, "suburb", 12, 10],
          "text-font": ["Noto Sans Regular"] },
        paint: { "text-color": "#333333", "text-halo-color": "#ffffff", "text-halo-width": 2 } },
    ],
  } as StyleSpecification;
}
