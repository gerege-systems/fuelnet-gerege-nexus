/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// The fuel distribution network: the stations an organisation operates.

import { request } from "./client";

export type FuelStation = {
  id: string;
  name: string;
  brand: string;
  brand_label: string;
  lat: number;
  lon: number;
  aimag: string;
  district: string;
  address: string;
  phone: string;
  opening_hours: string;
  total_pumps: number;
  active_pumps: number;
  /** No real source until the redemption stream exists — 0 everywhere for now. */
  current_queue_count: number;
  status: string;
  is_voucher_enabled: boolean;
  fuel_type_count: number;
};

export type FuelStationList = {
  stations: FuelStation[];
  count: number;
};

/** One fuel a station sells, as a citizen sees it. No litres — see the handler. */
export type PublicFuel = {
  type: string;
  label: string;
  price_mnt: number;
  status: string;
  /** Percentage of tank, or null where nobody has reported a tank size. */
  stock_percent: number | null;
};

/** A station as somebody looking for fuel sees it. */
export type PublicStation = {
  id: string;
  name: string;
  brand: string;
  brand_label: string;
  lat: number;
  lon: number;
  aimag: string;
  district: string;
  address: string;
  opening_hours: string;
  status: string;
  is_voucher_enabled: boolean;
  fuels: PublicFuel[];
  /** The fullest tank on the forecourt, or null when none has a reported size. */
  stock_percent: number | null;
};

export type PublicStationList = {
  stations: PublicStation[];
  count: number;
  /** The viewport held more than one answer carries. Zoom in for the rest. */
  truncated: boolean;
};

/** A map viewport, in the order GeoJSON and every mapping client use it. */
export type BBox = { minLon: number; minLat: number; maxLon: number; maxLat: number };

/** A tanker on its way to a forecourt, as somebody waiting for fuel sees it. */
export type PublicTrip = {
  id: string;
  trip_code: string;
  tanker_plate: string;
  brand: string;
  fuel_type: string;
  fuel_label: string;
  from_depot: string;
  to_station: string;
  to_station_id: string | null;
  status: string;
  lat: number;
  lon: number;
  heading: number;
  /** "device" when a tracker reported this, "schedule" when it is where the timetable says. */
  position_source: "device" | "schedule";
  progress_percent: number;
  eta_minutes: number | null;
  eta_at: string | null;
  /** When it left. With `eta_at` and `route`, a client can animate between polls. */
  departed_at: string;
  /** The road it is taking, as [[lon,lat], …]. Empty when the router was down. */
  route: Array<[number, number]>;
};

export type PublicTripList = { trips: PublicTrip[]; count: number };

/** What a citizen has left of today's ration. */
export type Entitlement = {
  date: string;
  granted_mnt: number;
  used_mnt: number;
  remaining_mnt: number;
};

/** A claim against the day's ration, redeemable at any pump. */
export type Voucher = {
  id: string;
  amount_mnt: number;
  fuel_type: string;
  fuel_label: string;
  qr_token: string;
  status: string;
  expires_at: string;
  created_at: string;
  intended_station?: string;
  redeemed_at?: string | null;
};

export const fuelApi = {
  /** The stations this organisation operates. Requires a session. */
  listFuelStations: () => request<FuelStationList>("/fuel/stations"),

  /**
   * The stations inside a map viewport, across every operator.
   *
   * No session: a person looking for fuel is not a member of anything. Rate
   * limited server-side at 60/min, so a caller that refires on every pixel of
   * a drag will be turned away — debounce on `moveend`, not on `move`.
   */
  publicFuelStations: (box: BBox) =>
    request<PublicStationList>(
      `/fuel/public/stations?bbox=${box.minLon},${box.minLat},${box.maxLon},${box.maxLat}`,
    ),

  /**
   * Tankers currently on the road, everywhere.
   *
   * No viewport: what makes a delivery interesting is where it is *going*, and
   * somebody waiting at a forecourt wants to know one is coming whether the
   * lorry is on the ring road or still in Darkhan.
   */
  publicFuelTrips: () => request<PublicTripList>("/fuel/public/trips"),

  /** Today's ration. Needs a session; a citizen signs in with eID. */
  myFuelEntitlement: () => request<Entitlement>("/fuel/me/entitlement"),

  /** Today's vouchers, newest first. */
  myFuelVouchers: () => request<{ vouchers: Voucher[]; count: number }>("/fuel/me/vouchers"),

  /**
   * Draw an amount out of today's ration.
   *
   * `intended_station_id` is a signal, not a commitment: the voucher is good at
   * any pump, and naming a forecourt only helps the queue estimate for it.
   */
  issueFuelVoucher: (body: {
    amount_mnt: number;
    fuel_type: string;
    intended_station_id?: string;
  }) =>
    request<{ voucher: Voucher; entitlement: Entitlement }>("/fuel/me/vouchers", {
      method: "POST",
      body: JSON.stringify(body),
    }),
};
