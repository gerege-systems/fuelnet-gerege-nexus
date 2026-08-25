/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

/**
 * The pin a filling station is drawn as.
 *
 * A teardrop in the operator's colour with a fuel pump on it. It replaced a
 * coloured circle, which said "something is here" and left the reader to work
 * out what — on a map that already carries POI dots, a plain circle is one more
 * dot among them.
 *
 * # Why images rather than one icon tinted per feature
 *
 * MapLibre cannot recolour an icon from a data expression: `icon-color` applies
 * only to SDF images, and an SDF is single-channel, so the white pump on a
 * coloured ground could not survive it. One image per colour is the way to keep
 * both the brand colour and the glyph, and there are six of them — five
 * operators with more than forty stations, and everybody else.
 *
 * They are generated here rather than shipped as files. A PNG per brand in
 * /public would be six assets to redraw the day the palette moves, and the
 * palette is a product decision that will move.
 */

/** Operator → pin colour. The key is `fuel_stations.brand`. */
export const BRAND_PIN_COLOURS: Record<string, string> = {
  petrovis: "#0064E1",
  shunkhlai: "#E11D48",
  "magnai-trade": "#F59E0B",
  "sod-mongol": "#059669",
  "tes-petroleum": "#7C3AED",
};

/** Everything else, including the third of the register recorded as "other". */
export const FALLBACK_PIN_COLOUR = "#64748B";

/** The image name a brand's pin is registered under. */
export function pinImageID(brand: string): string {
  return `fuel-pin-${brand in BRAND_PIN_COLOURS ? brand : "other"}`;
}

/** Drawn at twice the display size so the pin stays sharp on a retina screen. */
const PIN_WIDTH = 44;
const PIN_HEIGHT = 58;
const SCALE = 2;

/**
 * The pump glyph, from lucide's `fuel` — the same icon set the rest of the
 * interface uses, so a station on the map and a station in a list are the same
 * shape. Stroked rather than filled, on a 24×24 grid.
 */
const PUMP_PATHS = [
  "M3 22h12",
  "M4 9h10",
  "M14 22V4a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v18",
  "M14 13h2a2 2 0 0 1 2 2v2a2 2 0 0 0 2 2 2 2 0 0 0 2-2V9.83a2 2 0 0 0-.59-1.42L18 5",
];

function pinSVG(colour: string): string {
  const glyph = PUMP_PATHS.map(
    (d) => `<path d="${d}" fill="none" stroke="#fff" stroke-width="2.2"
             stroke-linecap="round" stroke-linejoin="round"/>`,
  ).join("");

  return `<svg xmlns="http://www.w3.org/2000/svg" width="${PIN_WIDTH * SCALE}" height="${
    PIN_HEIGHT * SCALE
  }" viewBox="0 0 ${PIN_WIDTH} ${PIN_HEIGHT}">
    <path d="M22 1.5C10.7 1.5 1.5 10.4 1.5 21.4c0 13.5 17.6 32.3 19.3 34.1a1.7 1.7 0 0 0 2.4 0c1.7-1.8 19.3-20.6 19.3-34.1C42.5 10.4 33.3 1.5 22 1.5Z"
          fill="${colour}" stroke="#ffffff" stroke-width="3"/>
    <g transform="translate(9.5,8) scale(0.92)">${glyph}</g>
  </svg>`;
}

/** One decoded pin, ready for `map.addImage`. */
function loadPin(colour: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image(PIN_WIDTH * SCALE, PIN_HEIGHT * SCALE);
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error("could not draw the station pin"));
    // A data URL rather than a blob: nothing has to be revoked, and the image
    // is a few hundred bytes.
    image.src = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(pinSVG(colour))}`;
  });
}

/**
 * Register every pin on a map, once its style is loaded.
 *
 * `pixelRatio: SCALE` is what tells MapLibre the bitmap is double-size, so the
 * pin lands at 44×58 CSS pixels rather than twice that.
 */
export async function addStationPins(map: {
  hasImage: (id: string) => boolean;
  addImage: (id: string, image: HTMLImageElement, options?: { pixelRatio?: number }) => void;
}): Promise<void> {
  const entries: Array<[string, string]> = [
    ...Object.entries(BRAND_PIN_COLOURS),
    ["other", FALLBACK_PIN_COLOUR],
  ];

  await Promise.all(
    entries.map(async ([brand, colour]) => {
      const id = pinImageID(brand);
      if (map.hasImage(id)) return;
      map.addImage(id, await loadPin(colour), { pixelRatio: SCALE });
    }),
  );
}

/**
 * The expression choosing a pin for a feature.
 *
 * Written out as a literal because MapLibre types `match` as a tuple, and a
 * value spread from a table widens to string[] and stops compiling.
 */
export const PIN_IMAGE_MATCH = [
  "match",
  ["get", "brand"],
  "petrovis", pinImageID("petrovis"),
  "shunkhlai", pinImageID("shunkhlai"),
  "magnai-trade", pinImageID("magnai-trade"),
  "sod-mongol", pinImageID("sod-mongol"),
  "tes-petroleum", pinImageID("tes-petroleum"),
  pinImageID("other"),
];

// ─────────────────────────────────────────────────────────────────────────────
// The tanker
// ─────────────────────────────────────────────────────────────────────────────

/** The image name a tanker is drawn under. */
export const TANKER_IMAGE_ID = "fuel-tanker";

/** Long and narrow, the proportions of an articulated lorry from above. */
const TANKER_W = 26;
const TANKER_H = 52;

/**
 * A tanker lorry seen from above, nose up.
 *
 * From above because the map rotates and tilts, and the symbol is rotated to
 * the vehicle's heading: anything drawn in profile would be driving sideways
 * half the time. Nose up because a rotation of zero has to mean something.
 *
 * It is a vehicle rather than a marker — cab, tank barrel with its ribs, wheels
 * proud of the body — so that on a street it reads as traffic and not as
 * another pin. The first version was a circle with a lorry glyph inside it and
 * looked like a badge somebody had dropped on the map.
 */
function tankerSVG(): string {
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${TANKER_W * 2}" height="${
    TANKER_H * 2
  }" viewBox="0 0 ${TANKER_W} ${TANKER_H}">
    <g stroke="#0b1220" stroke-width="1.1" stroke-linejoin="round">
      <!-- wheels, drawn first so the body sits over them -->
      <rect x="0.5"  y="12" width="4" height="7"  rx="1.6" fill="#1e293b"/>
      <rect x="21.5" y="12" width="4" height="7"  rx="1.6" fill="#1e293b"/>
      <rect x="0.5"  y="33" width="4" height="9"  rx="1.6" fill="#1e293b"/>
      <rect x="21.5" y="33" width="4" height="9"  rx="1.6" fill="#1e293b"/>

      <!-- cab -->
      <rect x="3.5" y="3" width="19" height="14" rx="3.5" fill="#f8fafc"/>
      <!-- windscreen, at the nose, so direction is readable at a glance -->
      <path d="M6 6.5h14v4.2H6z" fill="#93c5fd" stroke="none"/>

      <!-- tank -->
      <rect x="2.5" y="18" width="21" height="30" rx="6" fill="#e2e8f0"/>
      <g stroke="#94a3b8" stroke-width="1">
        <line x1="4" y1="26" x2="22" y2="26"/>
        <line x1="4" y1="33" x2="22" y2="33"/>
        <line x1="4" y1="40" x2="22" y2="40"/>
      </g>
      <!-- hazard stripe: what makes it a fuel lorry rather than any lorry -->
      <rect x="2.5" y="43.5" width="21" height="4.5" rx="2" fill="#f59e0b" stroke="none"/>
    </g>
  </svg>`;
}

/** Register the tanker image on a map, once its style is loaded. */
export async function addTankerImage(map: {
  hasImage: (id: string) => boolean;
  addImage: (id: string, image: HTMLImageElement, options?: { pixelRatio?: number }) => void;
}): Promise<void> {
  if (map.hasImage(TANKER_IMAGE_ID)) return;
  const image = await new Promise<HTMLImageElement>((resolve, reject) => {
    const img = new Image(TANKER_W * 2, TANKER_H * 2);
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error("could not draw the tanker"));
    img.src = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(tankerSVG())}`;
  });
  map.addImage(TANKER_IMAGE_ID, image, { pixelRatio: 2 });
}
