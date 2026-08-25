/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

import type { Metadata } from "next";

import FuelMap from "@/components/fuel/FuelMap";

/**
 * Where to buy fuel — the citizen's map.
 *
 * A page of its own rather than the front page. `/` argues the platform's case
 * to somebody deciding whether to adopt it; this is for a person standing next
 * to a car who wants to know which forecourt has petrol, and the two audiences
 * do not want the same screen.
 *
 * It is on the public list in lib/publicPath.mjs, so Layout renders it without
 * the tenant chrome: no sidebar, no app rail, no organisation switcher. A
 * driver is not administering anything, and on a first visit holds no session
 * at all — the sign-in prompt inside the map is what offers one.
 */

export const metadata: Metadata = {
  title: "Газрын зураг · FuelNet",
  description:
    "Ойролцоох шатахуун түгээх станцууд, тэдгээрийн үнэ, нөөцийн түвшин, замд яваа цистернүүд.",
};

export default function FuelMapPage() {
  return <FuelMap />;
}
