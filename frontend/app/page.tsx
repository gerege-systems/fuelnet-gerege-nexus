/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

import type { Metadata } from "next";

import FuelMap from "@/components/fuel/FuelMap";

/**
 * The front page: where to buy fuel.
 *
 * Upstream serves the platform's marketing landing here — Hero, Capabilities,
 * Architecture, Trust — composed from `LANDING_SECTIONS`. That page argues for
 * the platform to somebody deciding whether to adopt one. This deployment's
 * visitor has already arrived, is standing next to a car, and wants to know
 * which forecourt has petrol; an argument about modular monoliths is not what
 * they came for.
 *
 * This is one of the handful of shared files this fork diverges in, and the
 * divergence is the point of the fork. See BENZIN_NEXUS_PORT_PLAN.md §8.3 for
 * the full list and how merges are kept cheap.
 *
 * The operator-facing surfaces are untouched and reached as before:
 * `/login` for a session, `/fuel` for the station register, `/apps` for the
 * store.
 */

export const metadata: Metadata = {
  title: "Шатахуун хаана байна",
  description:
    "Ойролцоох шатахуун түгээх станцууд, тэдгээрийн үнэ болон нөөцийн түвшин — бодит цагийн зураг.",
};

export default function HomePage() {
  return <FuelMap />;
}
