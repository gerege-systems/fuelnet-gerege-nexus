/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

/**
 * Where a lorry is, right now, worked out in the browser.
 *
 * The server sends each run's road, the time it left and the time it is due.
 * That is everything the server itself uses, so the client can compute the same
 * answer for any instant — including the instants between two polls.
 *
 * Which is the whole point. Polling every fifteen seconds and drawing whatever
 * came back makes vehicles teleport fifteen seconds' worth of road at a time; a
 * poll is for *news* — a run finished, a new one dispatched, a tracker reported
 * a real position — and motion belongs on the animation frame.
 */

import type { PublicTrip } from "@/lib/api";

export type Placed = {
  trip: PublicTrip;
  lat: number;
  lon: number;
  heading: number;
  progress: number;
};

/** How far through its run a trip is at a given moment, 0..1. */
export function progressAt(trip: PublicTrip, now: number): number {
  if (!trip.eta_at) return 0;
  const departed = Date.parse(trip.departed_at);
  const due = Date.parse(trip.eta_at);
  if (!Number.isFinite(departed) || !Number.isFinite(due) || due <= departed) return 1;
  return Math.min(1, Math.max(0, (now - departed) / (due - departed)));
}

/**
 * The point a fraction of the way along a polyline, by distance, and the heading
 * of the segment it lands on.
 *
 * By distance rather than by vertex, for the reason the server's copy of this
 * gives: a router puts points where the road bends, so a junction-heavy stretch
 * has many and a highway has two. Stepping vertex by vertex crawls through town
 * and then jumps across the steppe.
 */
export function alongRoute(
  route: Array<[number, number]>,
  progress: number,
): { lat: number; lon: number; heading: number } {
  if (route.length < 2) {
    const [lon, lat] = route[0] ?? [0, 0];
    return { lat, lon, heading: 0 };
  }

  const lengths: number[] = [];
  let total = 0;
  for (let i = 0; i < route.length - 1; i++) {
    const dLon = (route[i + 1][0] - route[i][0]) * Math.cos((route[i][1] * Math.PI) / 180);
    const dLat = route[i + 1][1] - route[i][1];
    const length = Math.hypot(dLon, dLat);
    lengths.push(length);
    total += length;
  }
  if (total === 0) return { lat: route[0][1], lon: route[0][0], heading: 0 };

  const target = progress * total;
  let walked = 0;
  for (let i = 0; i < lengths.length; i++) {
    if (walked + lengths[i] >= target || i === lengths.length - 1) {
      const within = lengths[i] > 0 ? Math.min(1, Math.max(0, (target - walked) / lengths[i])) : 0;
      return {
        lon: route[i][0] + (route[i + 1][0] - route[i][0]) * within,
        lat: route[i][1] + (route[i + 1][1] - route[i][1]) * within,
        heading: bearing(route[i][1], route[i][0], route[i + 1][1], route[i + 1][0]),
      };
    }
    walked += lengths[i];
  }
  const last = route[route.length - 1];
  return { lat: last[1], lon: last[0], heading: 0 };
}

function bearing(lat1: number, lon1: number, lat2: number, lon2: number): number {
  const f1 = (lat1 * Math.PI) / 180;
  const f2 = (lat2 * Math.PI) / 180;
  const dl = ((lon2 - lon1) * Math.PI) / 180;
  const y = Math.sin(dl) * Math.cos(f2);
  const x = Math.cos(f1) * Math.sin(f2) - Math.sin(f1) * Math.cos(f2) * Math.cos(dl);
  return (((Math.atan2(y, x) * 180) / Math.PI) + 360) % 360;
}

/**
 * Every trip placed for one instant.
 *
 * A run whose tracker has reported stays where the report put it: a real
 * position beats a computed one, and sliding it along a schedule would be
 * inventing movement the vehicle did not make.
 */
export function placeTrips(trips: PublicTrip[], now: number): Placed[] {
  return trips.map((trip) => {
    const progress = progressAt(trip, now);
    if (trip.position_source === "device" || trip.route.length < 2) {
      return { trip, lat: trip.lat, lon: trip.lon, heading: trip.heading, progress };
    }
    const at = alongRoute(trip.route, progress);
    return { trip, ...at, progress };
  });
}
