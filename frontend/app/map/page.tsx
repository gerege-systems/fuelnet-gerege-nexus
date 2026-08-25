/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

import type { Metadata } from "next";

import FuelMap from "@/components/fuel/FuelMap";
import FuelNetHeader from "@/components/fuelnet/FuelNetHeader";

/**
 * Where to buy fuel — the citizen's map.
 *
 * The site's header above it and the map filling everything below. The header
 * rather than the whole FuelNetShell, because the shell ends in a footer and a
 * footer under a full-bleed map is a scroll nobody wants: the map is the page.
 *
 * It carries the way in, which is why the map itself no longer needs to. The
 * panel that used to run down the left held the sign-in button, the entitlement
 * and the voucher list; on a phone — where a driver actually opens this — it
 * was the entire screen, with the map pushed off the side.
 */

export const metadata: Metadata = {
  title: "Газрын зураг · FuelNet",
  description:
    "Ойролцоох шатахуун түгээх станцууд, тэдгээрийн үнэ, нөөцийн түвшин, замд яваа цистернүүд.",
};

export default function FuelMapPage() {
  return (
    <div className="flex h-dvh flex-col">
      <FuelNetHeader />
      <div className="relative min-h-0 flex-1">
        <FuelMap />
      </div>
    </div>
  );
}
