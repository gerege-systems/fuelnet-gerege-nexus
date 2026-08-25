"use client";

/**
 * Where to buy fuel — the citizen's map, and this deployment's front page.
 *
 * Rendered by app/page.tsx at `/`, which components/Layout.tsx already treats as
 * public: Layout returns the children and nothing else, so there is no sidebar,
 * no app rail and no organisation switcher around it. Those belong to somebody
 * administering a company. A driver looking for petrol is not administering
 * anything and, on a first visit, has no session at all.
 *
 * It replaced the platform's marketing landing page. That page argues for the
 * platform to somebody choosing one; this deployment's visitor has already
 * arrived and wants to know where the fuel is.
 *
 * # Why a GeoJSON source and not markers
 *
 * Up to 300 stations in a viewport. Three hundred DOM markers is three hundred
 * absolutely-positioned elements that the browser re-lays-out on every frame of
 * a pan; a GeoJSON source with a circle layer is drawn by the GPU in one pass
 * and stays smooth on a phone. It also lets the brand colour come from a data
 * expression rather than from JavaScript deciding per marker.
 */

import { useCallback, useEffect, useRef, useState } from "react";

import CitizenRail from "@/components/fuel/CitizenRail";
import StationSheet from "@/components/fuel/StationSheet";
// Named imports, not a default one: maplibre-gl ships ESM from v5 and its
// module has no default export, so `import maplibregl from` builds against the
// CommonJS shim under webpack and fails outright under Turbopack.
import {
  GeolocateControl,
  Map as MapLibreMap,
  NavigationControl,
  Popup,
  setWorkerUrl,
  type MapLayerMouseEvent,
  type GeoJSONSource,
} from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";

import { api, type PublicStation, type PublicTrip } from "@/lib/api";
import { fuelMapStyle, isCovered, ULAANBAATAR } from "@/lib/fuel/mapStyle";
import { addStationPins, addTankerImage, PIN_IMAGE_MATCH, TANKER_IMAGE_ID } from "@/lib/fuel/stationPin";
import { createTankerLayer, TANKER_MODEL_MINZOOM } from "@/lib/fuel/tankerLayer";
import { placeTrips } from "@/lib/fuel/tripMotion";
import { useI18n } from "@/lib/i18n";

const SOURCE = "stations";
const TRIPS = "trips";
const ROUTE = "trip-route";
const TRIP_MODELS = "trip-models";

/**
 * Tell MapLibre where its worker lives.
 *
 * It parses tiles on a module worker whose url it derives from the library's
 * own `import.meta.url`. Under Turbopack that resolves to a path the dev server
 * answers with HTML, the browser rejects it on MIME type, and the map renders
 * nothing — with no error on the map object, because the failure is inside the
 * worker's loader rather than in anything MapLibre awaits. The symptom is a
 * blank canvas and one console line about a "non-JavaScript MIME type".
 *
 * scripts/copy-maplibre-worker.mjs puts the file under /public on predev and
 * prebuild, so this path is served as a static asset with the right type.
 */
setWorkerUrl("/maplibre/maplibre-gl-worker.mjs");

/**
 * What a lorry carries into the map, for both the icon and the solid body.
 *
 * One function because the two sources must agree: they are the same vehicles
 * at two zoom levels, and a tap has to say the same thing whichever one caught
 * it. Feature properties are flat, so the route polyline rides as JSON.
 */
function tripProperties(t: PublicTrip): Record<string, string | number> {
  return {
    plate: t.tanker_plate,
    fuelLabel: t.fuel_label,
    fromDepot: t.from_depot,
    toStation: t.to_station,
    heading: t.heading,
    etaMinutes: t.eta_minutes ?? -1,
    progress: t.progress_percent,
    positionSource: t.position_source,
    route: JSON.stringify(t.route ?? []),
  };
}

function toFeatureCollection(stations: PublicStation[]): GeoJSON.FeatureCollection {
  return {
    type: "FeatureCollection",
    features: stations.map((s) => ({
      type: "Feature",
      geometry: { type: "Point", coordinates: [s.lon, s.lat] },
      properties: {
        id: s.id,
        name: s.name,
        brand: s.brand,
        brandLabel: s.brand_label || s.brand,
        address: s.address,
        openingHours: s.opening_hours,
        // -1 stands for "nobody has reported a tank size". A real percentage is
        // 0..100, so the sentinel cannot collide with one, and MapLibre feature
        // properties have no null to use instead.
        stock: s.stock_percent ?? -1,
        // MapLibre feature properties are flat, so the fuels ride as JSON and
        // are parsed back when a popup needs them.
        fuels: JSON.stringify(s.fuels ?? []),
      },
    })),
  };
}

export default function FuelMap() {
  const { t } = useI18n();
  const container = useRef<HTMLDivElement | null>(null);
  const map = useRef<InstanceType<typeof MapLibreMap> | null>(null);
  const [count, setCount] = useState<number | null>(null);
  const [truncated, setTruncated] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tripCount, setTripCount] = useState(0);
  // The station whose panel is open. Held in React rather than drawn into a
  // MapLibre popup because the panel has a form in it — see StationSheet.
  const [selected, setSelected] = useState<PublicStation | null>(null);
  // The last answer from the server, so a tap can find the whole station rather
  // than the flattened properties MapLibre carries on a feature.
  const stationsByID = useRef<Map<string, PublicStation>>(new Map());
  // The three.js layer keeps its own scene, so the fleet is handed to it rather
  // than pushed through a GeoJSON source like everything else on this map.
  const tankerLayer = useRef<ReturnType<typeof createTankerLayer> | null>(null);
  // The last answer from the server. The animation frame reads it; the poll
  // replaces it. Nothing else needs to know a poll happened.
  const tripsRef = useRef<PublicTrip[]>([]);

  /** Read the viewport, ask the server, hand the answer to the source. */
  const refresh = useCallback(async () => {
    const m = map.current;
    if (!m) return;
    const b = m.getBounds();
    try {
      const result = await api.publicFuelStations({
        minLon: b.getWest(),
        minLat: b.getSouth(),
        maxLon: b.getEast(),
        maxLat: b.getNorth(),
      });
      const source = m.getSource(SOURCE) as GeoJSONSource | undefined;
      source?.setData(toFeatureCollection(result.stations));
      stationsByID.current = new Map(result.stations.map((s) => [s.id, s]));
      setCount(result.count);
      setTruncated(result.truncated);
      setError(null);
    } catch (err) {
      // A 429 is the rate limiter, not a fault: the map stays as it was and the
      // next settled viewport asks again.
      setError(err instanceof Error ? err.message : String(err));
    }
  }, []);

  /**
   * Deliveries on the road.
   *
   * Separate from the station refresh and on its own timer: stations change
   * when an operator edits one, tankers change continuously, and tying the two
   * would mean either stale lorries or a station query every fifteen seconds.
   */
  const refreshTrips = useCallback(async () => {
    const m = map.current;
    if (!m) return;
    try {
      const result = await api.publicFuelTrips();
      // Stored, not drawn. A poll is for news — a run finished, a new one was
      // dispatched, a tracker reported a real position. Where the lorries are
      // between polls is worked out on the animation frame, which is what makes
      // them move rather than jump every fifteen seconds.
      tripsRef.current = result.trips;
      setTripCount(result.count);
    } catch {
      // A delivery estimate is a nicety. Losing it must not take the map with it.
    }
  }, []);

  useEffect(() => {
    if (!container.current || map.current) return;

    const m = new MapLibreMap({
      container: container.current,
      style: fuelMapStyle(),
      center: [ULAANBAATAR.lon, ULAANBAATAR.lat],
      zoom: 12,
      // Tilted a little from the start, so the buildings this style extrudes
      // are visible as buildings and somebody knows the view can be moved at
      // all. Flat-on, a 3D map looks exactly like a 2D one.
      pitch: 45,
      bearing: -12,
      maxPitch: 75,
      attributionControl: { compact: true },
    });
    map.current = m;

    // MapLibre reports a broken style or an unreachable tile here and nowhere
    // else. Without this the map simply draws nothing, which is the same thing
    // a working map over empty tiles looks like.
    m.on("error", (event) => {
      setError(event.error?.message ?? "map error");
    });

    // With the compass: it is the control that says the map turns, and it is
    // how somebody gets back to north after it has. `visualizePitch` puts the
    // tilt on the needle as well, so the two are one control rather than a
    // gesture nobody discovers.
    m.addControl(new NavigationControl({ showCompass: true, visualizePitch: true }), "bottom-right");
    m.addControl(
      new GeolocateControl({ trackUserLocation: false, showAccuracyCircle: true }),
      "bottom-right",
    );

    m.on("load", async () => {
      m.addSource(SOURCE, { type: "geojson", data: toFeatureCollection([]) });

      // A pin per station, not a circle. The image is chosen from the feature's
      // brand, which is why there is one image per colour — see lib/fuel/stationPin.ts.
      await addStationPins(m);
      m.addLayer({
        id: "station-pin",
        type: "symbol",
        source: SOURCE,
        layout: {
          "icon-image": PIN_IMAGE_MATCH as never,
          // Anchored at the point of the teardrop, so the pin marks the station
          // rather than hovering above it.
          "icon-anchor": "bottom",
          "icon-size": ["interpolate", ["linear"], ["zoom"], 8, 0.45, 13, 0.7, 17, 1],
          // Stations cluster along a road; letting MapLibre drop the ones that
          // collide would hide exactly the choice a driver is making.
          "icon-allow-overlap": true,
        },
      });

      // How full the fullest tank is, as a dot on the pin's shoulder.
      //
      // Green / amber / red, and grey where no tank here has a reported size —
      // an unknown level and an empty one send a driver to opposite places, so
      // they must not look alike. Drawn as its own circle layer rather than
      // baked into the pin image: six brands times four levels would be
      // twenty-four images to redraw whenever either scale moves.
      m.addLayer({
        id: "station-level",
        type: "circle",
        source: SOURCE,
        minzoom: 10,
        paint: {
          "circle-radius": ["interpolate", ["linear"], ["zoom"], 10, 3.5, 17, 6],
          "circle-color": [
            "case",
            ["<", ["get", "stock"], 0], "#94a3b8",
            ["<", ["get", "stock"], 15], "#dc2626",
            ["<", ["get", "stock"], 40], "#f59e0b",
            "#16a34a",
          ],
          "circle-stroke-width": 1.5,
          "circle-stroke-color": "#ffffff",
          // Onto the pin's upper right. The pin is anchored at its point, so the
          // offset is measured up from the station itself.
          "circle-translate": ["interpolate", ["linear"], ["zoom"], 10, ["literal", [9, -30]], 17, ["literal", [13, -44]]],
          "circle-translate-anchor": "viewport",
        },
      });

      // The road a selected delivery is taking. Empty until somebody taps one:
      // sixteen routes at five hundred points each is a lot of line to draw for
      // a question nobody asked.
      m.addSource(ROUTE, { type: "geojson", data: { type: "FeatureCollection", features: [] } });
      m.addLayer({
        id: "trip-route-casing",
        type: "line",
        source: ROUTE,
        layout: { "line-cap": "round", "line-join": "round" },
        paint: {
          "line-color": "#ffffff",
          "line-width": ["interpolate", ["linear"], ["zoom"], 8, 5, 16, 11],
          "line-opacity": 0.9,
        },
      });
      m.addLayer({
        id: "trip-route-line",
        type: "line",
        source: ROUTE,
        layout: { "line-cap": "round", "line-join": "round" },
        paint: {
          "line-color": "#0f172a",
          "line-width": ["interpolate", ["linear"], ["zoom"], 8, 2.5, 16, 6],
          // Dashed: the lorry is on the road now, the line is where it is going.
          "line-dasharray": [2, 1.6],
        },
      });

      // Tankers. Above the stations, because a vehicle passing over a forecourt
      // is the one that moved and the one worth noticing.
      await addTankerImage(m);
      m.addSource(TRIPS, { type: "geojson", data: { type: "FeatureCollection", features: [] } });
      // Solid lorries, close in — real models rather than extruded prisms.
      // Above the same zoom the style starts extruding buildings, so the two
      // kinds of object appear together and neither stands beside a flat
      // version of the other.
      const models = createTankerLayer(TRIP_MODELS);
      tankerLayer.current = models;
      m.addLayer(models);

      m.addLayer({
        id: "trip-tanker",
        type: "symbol",
        source: TRIPS,
        layout: {
          "icon-image": TANKER_IMAGE_ID,
          "icon-size": ["interpolate", ["linear"], ["zoom"], 6, 0.4, 12, 0.6, 17, 0.85],
          "icon-allow-overlap": true,
          // Rotated to the run's heading, in map space: the lorry points where
          // it is going even after the map itself is rotated.
          "icon-rotate": ["get", "heading"],
          "icon-rotation-alignment": "map",
        },
        paint: {
          // Faded out where the models take over, rather than removed: the
          // layer is still what a tap lands on, and a layer with no features
          // rendered is a layer chi cannot hit-test.
          "icon-opacity": [
            "interpolate", ["linear"], ["zoom"],
            TANKER_MODEL_MINZOOM - 0.5, 1,
            TANKER_MODEL_MINZOOM, 0,
          ],
        },
      });

      m.on("mouseenter", "trip-tanker", () => (m.getCanvas().style.cursor = "pointer"));
      m.on("mouseleave", "trip-tanker", () => (m.getCanvas().style.cursor = ""));
      // A custom layer holds no features, so it cannot be clicked. The icon
      // layer stays mounted underneath it as the tap target — invisible above
      // the model zoom, and still the thing MapLibre hit-tests.
      // One handler on both layers: they are the same vehicles, and which one
      // catches the tap is only a question of how far zoomed in somebody is.
      const openTrip = (event: MapLayerMouseEvent) => {
        const feature = event.features?.[0];
        if (!feature) return;
        const p = feature.properties as Record<string, string | number>;
        const eta = Number(p.etaMinutes);
        const etaText = eta < 0 ? "—" : eta === 0 ? "ирлээ" : `${eta} мин`;
        // Said out loud: an interpolated point is not a tracker reading, and a
        // map that showed them identically would claim a precision it has not got.
        const sourceNote =
          p.positionSource === "device" ? "GPS-ээр" : "хуваариар тооцоолсон";

        // Show where it is going. The route came down with the trip, so this
        // draws immediately rather than asking the server on a tap.
        const source = m.getSource(ROUTE) as GeoJSONSource | undefined;
        let route: Array<[number, number]> = [];
        try {
          route = JSON.parse(String(p.route || "[]"));
        } catch {
          /* a delivery with no drawable route still gets its popup */
        }
        source?.setData({
          type: "FeatureCollection",
          features: route.length
            ? [{ type: "Feature", properties: {}, geometry: { type: "LineString", coordinates: route } }]
            : [],
        });

        // `Popup.on` answers a Subscription rather than the popup, so the
        // listener is attached on its own line instead of in the chain.
        const tripPopup = new Popup({ closeButton: true, maxWidth: "280px" });
        tripPopup.on("close", () => {
          // The line belongs to the popup that asked for it.
          (m.getSource(ROUTE) as GeoJSONSource | undefined)?.setData({
            type: "FeatureCollection",
            features: [],
          });
        });
        tripPopup
          // The tap position rather than the feature's own: a polygon has no
          // single point, and a click on a lorry is already on the lorry.
          .setLngLat(event.lngLat)
          .setHTML(
            `<div style="font:14px/1.45 system-ui">
               <div style="font-weight:600">${p.plate} · ${p.fuelLabel}</div>
               <div style="color:#64748b;font-size:12px;margin:.25rem 0 .5rem">
                 ${p.fromDepot} → ${p.toStation || "ШТС"}
               </div>
               <div style="display:flex;justify-content:space-between"><span>Хүрэх</span><b>${etaText}</b></div>
               <div style="margin-top:.4rem;height:5px;border-radius:99px;background:#eef2f7;overflow:hidden">
                 <div style="width:${p.progress}%;height:100%;background:#0f172a"></div>
               </div>
               <div style="color:#94a3b8;font-size:11px;margin-top:.4rem">${sourceNote}</div>
             </div>`,
          )
          .addTo(m);
      };
      m.on("click", "trip-tanker", openTrip);

      m.on("mouseenter", "station-pin", () => (m.getCanvas().style.cursor = "pointer"));
      m.on("mouseleave", "station-pin", () => (m.getCanvas().style.cursor = ""));
      m.on("click", "station-pin", (event) => {
        const feature = event.features?.[0];
        if (!feature) return;
        const id = String((feature.properties as Record<string, string>).id);
        const station = stationsByID.current.get(id);
        if (station) setSelected(station);
      });

      void refresh();
      void refreshTrips();
    });

    // Tankers move whether or not the map does, so the *poll* is on a clock.
    // Twenty seconds: it is only asking whether the board changed.
    const tripTimer = window.setInterval(() => void refreshTrips(), 20000);

    // And the motion is on the frame. Each pass advances every vehicle along
    // its own road by however much wall-clock time has passed, so the fleet
    // moves continuously and at its own speed — a lorry on a trunk road pulls
    // away from one crossing town, because that is what their schedules say.
    let frame = 0;
    const animate = () => {
      frame = requestAnimationFrame(animate);
      const trips = tripsRef.current;
      if (trips.length === 0) return;

      const placed = placeTrips(trips, Date.now());

      const points = m.getSource(TRIPS) as GeoJSONSource | undefined;
      points?.setData({
        type: "FeatureCollection",
        features: placed.map((p) => ({
          type: "Feature",
          geometry: { type: "Point", coordinates: [p.lon, p.lat] },
          properties: { ...tripProperties(p.trip), heading: p.heading, progress: p.progress * 100 },
        })),
      });

      tankerLayer.current?.setTankers(
        placed.map((p) => ({ lat: p.lat, lon: p.lon, heading: p.heading })),
      );
    };
    frame = requestAnimationFrame(animate);

    // On the settled viewport, not on every frame of a drag: the server allows
    // sixty calls a minute and a pan fires `move` far more often than that.
    m.on("moveend", () => void refresh());

    // Centre on the visitor only where we have tiles. A position abroad — a
    // traveller, or a browser guessing badly — would open onto the background
    // colour and read as a broken page.
    if (typeof navigator !== "undefined" && navigator.geolocation) {
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          const { latitude, longitude } = pos.coords;
          if (isCovered(latitude, longitude)) m.jumpTo({ center: [longitude, latitude], zoom: 13 });
        },
        () => {
          /* refused or unavailable: Ulaanbaatar is a fine answer */
        },
        { enableHighAccuracy: true, timeout: 8000 },
      );
    }

    return () => {
      cancelAnimationFrame(frame);
      window.clearInterval(tripTimer);
      m.remove();
      map.current = null;
    };
  }, [refresh, refreshTrips, t]);

  return (
    // The height is an inline style rather than a utility class on purpose.
    // With `h-dvh` the wrapper measured zero high — every child here is
    // absolutely positioned, so nothing gives it intrinsic height, and MapLibre
    // left the canvas at the <canvas> element's own 300x150 default. An inline
    // value cannot be missing from a stylesheet, and `100vh` is behind it for
    // anything that does not know `dvh`.
    <div className="relative w-full" style={{ height: "100dvh", minHeight: "100vh" }}>
      <div ref={container} className="absolute inset-0" style={{ height: "100%" }} />


      <CitizenRail stationCount={count} tripCount={tripCount} />

      {error ? (
        <p className="pointer-events-auto absolute left-1/2 top-3 z-10 -translate-x-1/2 rounded-xl bg-red-50 px-4 py-2 text-sm text-red-700 ring-1 ring-red-200">
          {error}
        </p>
      ) : null}

      {selected ? <StationSheet station={selected} onClose={() => setSelected(null)} /> : null}
    </div>
  );
}
