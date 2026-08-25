/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

/**
 * The tankers, as models.
 *
 * A MapLibre custom layer: the map hands over its GL context and its
 * model-view-projection matrix once per frame, and three.js draws into the same
 * context. That is the only way a real mesh gets onto this map — `fill-extrusion`
 * draws prisms with vertical walls and nothing else, which is why the lorries
 * were briefly two grey boxes.
 *
 * # The coordinate system, and why there is an origin
 *
 * MapLibre's matrix maps *Mercator* coordinates to clip space: x east, y south,
 * z up, and the unit is the whole world rather than a metre — so a metre is
 * about 1.4e-8 of one.
 *
 * The obvious arrangement is to place each model at its Mercator coordinate and
 * scale it by that figure. It does not work, and the way it fails is worth
 * writing down because it looks like a broken model rather than a broken
 * transform: the object sits at ~0.79 with vertices ~1e-7 apart, GL interpolates
 * in float32, and eleven metres of lorry falls inside the rounding error of its
 * own position. What renders is a striped black smear.
 *
 * So three.js is given a world measured in **metres from an origin near the
 * fleet**, and the origin-to-Mercator step is folded into the camera's
 * projection matrix instead. Vertices are then metres apart at metre-scale
 * coordinates, which is what float32 is for. The origin follows the fleet, so
 * it never drifts far enough for the same problem to reappear.
 *
 * Local axes match Mercator's — x east, y south, z up — deliberately. Flipping y
 * to make it north would give a negative scale, which inverts every normal and
 * leaves the model lit from inside.
 *
 * The model is already z-up with its length on x, so it needs no axis fix.
 */

import {
  AmbientLight,
  DirectionalLight,
  Group,
  Matrix4,
  Scene,
  WebGLRenderer,
  Camera,
  type Object3D,
} from "three";
import { GLTFLoader } from "three/examples/jsm/loaders/GLTFLoader.js";
import { MercatorCoordinate, type CustomLayerInterface, type Map as MapLibreMap } from "maplibre-gl";

/** One vehicle to draw. */
export type TankerPlacement = {
  lat: number;
  lon: number;
  /** Compass degrees, clockwise from north. */
  heading: number;
};

const MODEL_URL = "/models/fuel-tanker.glb";

/**
 * The model is 8.7 m nose to tail. A rigid tanker is nearer eleven, and at map
 * scale the difference is the one between "a vehicle" and "a car".
 */
const SIZE_MULTIPLIER = 1.25;

/**
 * Below this the lorries are drawn as flat icons instead.
 *
 * The same zoom the style starts extruding buildings at: a model standing among
 * flat blocks would look pasted on, and an eleven-metre object is a fraction of
 * a pixel at city scale anyway.
 */
export const TANKER_MODEL_MINZOOM = 15;

export function createTankerLayer(id: string): CustomLayerInterface & {
  setTankers: (tankers: TankerPlacement[]) => void;
} {
  const scene = new Scene();
  const camera = new Camera();
  const fleet = new Group();
  scene.add(fleet);

  // Two lights and no shadows. The model carries no textures, so its shape is
  // read entirely off the shading: a single source leaves the shaded side flat
  // black, and ambient alone leaves it with no shape at all.
  const sun = new DirectionalLight(0xffffff, 2.2);
  sun.position.set(-0.6, -1, 1.4);
  scene.add(sun, new AmbientLight(0xffffff, 1.4));

  let renderer: WebGLRenderer | null = null;
  let prototype: Object3D | null = null;
  let pending: TankerPlacement[] = [];
  let map: MapLibreMap | null = null;

  /** Where the metre-space world has its zero. Recomputed with the fleet. */
  let origin = MercatorCoordinate.fromLngLat([106.9175, 47.9185], 0);
  let metre = origin.meterInMercatorCoordinateUnits();

  /** Put one clone per vehicle into the scene, in metres from the origin. */
  function place(tankers: TankerPlacement[]) {
    if (!prototype) return;
    fleet.clear();
    if (tankers.length === 0) {
      map?.triggerRepaint();
      return;
    }

    // The origin goes to the middle of the fleet, so no vehicle is more than a
    // few tens of kilometres from zero and every coordinate stays comfortably
    // inside float32.
    const midLon = tankers.reduce((sum, t) => sum + t.lon, 0) / tankers.length;
    const midLat = tankers.reduce((sum, t) => sum + t.lat, 0) / tankers.length;
    origin = MercatorCoordinate.fromLngLat([midLon, midLat], 0);
    metre = origin.meterInMercatorCoordinateUnits();

    for (const tanker of tankers) {
      const at = MercatorCoordinate.fromLngLat([tanker.lon, tanker.lat], 0);

      const model = prototype.clone(true);
      // Metres east and south of the origin.
      model.position.set((at.x - origin.x) / metre, (at.y - origin.y) / metre, 0);
      model.scale.setScalar(SIZE_MULTIPLIER);

      // The model's nose is +x, which is east — a bearing of 90°. y grows
      // southward, so a bearing θ wants the nose along (sin θ, −cos θ), and
      // rotating +x by θ−90° about z gives exactly that.
      model.rotation.z = ((tanker.heading - 90) * Math.PI) / 180;

      fleet.add(model);
    }
    map?.triggerRepaint();
  }

  return {
    id,
    type: "custom",
    renderingMode: "3d",

    onAdd(addedTo, gl) {
      map = addedTo;
      renderer = new WebGLRenderer({ canvas: addedTo.getCanvas(), context: gl, antialias: true });
      // MapLibre owns the canvas and everything already drawn on it. Clearing
      // or resizing would wipe the map out from under this layer.
      renderer.autoClear = false;

      new GLTFLoader().load(
        MODEL_URL,
        (gltf) => {
          prototype = gltf.scene;
          place(pending);
        },
        undefined,
        () => {
          // The map keeps working without the model — the flat icons below the
          // model zoom are still there, and a delivery is still on the list.
          console.warn("fuel: the tanker model could not be loaded");
        },
      );
    },

    render(_gl, args) {
      if (!renderer || !prototype || fleet.children.length === 0) return;
      // MapLibre v5 hands the matrix inside an argument object; older builds
      // passed it directly. Accepting both keeps this working across an upgrade
      // that would otherwise fail as a blank layer with no error.
      const matrix =
        (args as { defaultProjectionData?: { mainMatrix: number[] } }).defaultProjectionData
          ?.mainMatrix ?? (args as unknown as number[]);
      // clip ← mercator ← origin ← metres. The last two steps are what keep the
      // geometry itself at metre scale; see the note at the top of this file.
      camera.projectionMatrix = new Matrix4()
        .fromArray(matrix as number[])
        .multiply(new Matrix4().makeTranslation(origin.x, origin.y, origin.z))
        .multiply(new Matrix4().makeScale(metre, metre, metre));

      renderer.resetState();
      renderer.render(scene, camera);
    },

    onRemove() {
      fleet.clear();
      renderer?.dispose();
      renderer = null;
      map = null;
    },

    /** The current fleet. Held until the model arrives if it has not yet. */
    setTankers(tankers: TankerPlacement[]) {
      pending = tankers;
      place(tankers);
    },
  };
}
