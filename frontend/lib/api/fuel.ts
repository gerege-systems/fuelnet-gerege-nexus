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
  /** The grades this forecourt sells; they travel with it, like a depot's tanks. */
  fuels?: StationGrade[];
};

export type FuelStationList = {
  stations: FuelStation[];
  count: number;
};

/**
 * One grade at one forecourt: what it costs, how big the vessel is, and
 * whether it can be had.
 *
 * `current_stock_liters` is read-only everywhere. Litres in a tank are the sum
 * of what deliveries put there — see internal/apps/fuel/station.go — and the
 * honest way to say "we are out" is `status`, not a zero typed into a box.
 */
export type StationGrade = {
  fuel_type: string;
  fuel_label: string;
  price_mnt: number;
  tank_capacity_liters: number;
  current_stock_liters: number;
  status: string;
  last_reported_at: string | null;
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

/**
 * A consignment at, or past, the border.
 *
 * `declared_liters` is the figure everything downstream works in;
 * `declared_tons` is what the customs document says and is not reconciled
 * against it. Density moves with temperature, so the two never agree exactly,
 * and a screen that hid one of them would be choosing which to believe.
 */
export type Shipment = {
  id: string;
  declaration_no: string;
  border_port: string;
  origin_country: string;
  exporter: string;
  fuel_type: string;
  fuel_label: string;
  declared_liters: number;
  declared_tons: number | null;
  wagons: number;
  convoy_code: string;
  /** border_arrived · inspecting · cleared · in_transit · at_depot */
  status: string;
  lab_status: string;
  quality_cert_no: string;
  octane_tested: number | null;
  sulfur_ppm: number | null;
  entered_at: string;
  cleared_at: string | null;
  expected_at: string | null;
  /** Minted when customs clears the consignment, and empty until then. */
  batch_code?: string;
  note: string;
};

/** One vessel at a depot. */
export type Tank = {
  id: string;
  depot_id: string;
  tank_no: string;
  tank_type: string;
  fuel_type: string;
  fuel_label: string;
  capacity_liters: number;
  current_liters: number;
  fill_percent: number;
  temperature_c: number | null;
  density_kg_m3: number | null;
  safety_status: string;
  next_inspection_at: string | null;
};

/** A storage base, with its tanks. */
export type Depot = {
  id: string;
  name: string;
  brand: string;
  aimag: string;
  district: string;
  address: string;
  lat: number | null;
  lon: number | null;
  has_rail_siding: boolean;
  rail_station_code: string;
  status: string;
  tank_count: number;
  capacity_liters: number;
  current_liters: number;
  /** The whole base, tanks summed. Null when no tank is registered. */
  fill_percent: number | null;
  tanks?: Tank[];
};

/** One consignment unloaded into a tank. */
export type DepotReceipt = {
  id: string;
  depot_id: string;
  tank_id: string;
  shipment_id: string | null;
  batch_id: string | null;
  batch_code?: string;
  liters: number;
  received_at: string;
  tank_after_liters: number;
  tank_fill_percent: number;
};

export const fuelApi = {
  /** The stations this organisation operates. Requires a session. */
  listFuelStations: () => request<FuelStationList>("/fuel/stations"),

  /** Register a forecourt. Coordinates are required: the map has to draw it. */
  createFuelStation: (body: {
    name: string;
    brand?: string;
    brand_label?: string;
    aimag?: string;
    district?: string;
    address?: string;
    phone?: string;
    opening_hours?: string;
    lat: number;
    lon: number;
    total_pumps?: number;
    active_pumps?: number;
    is_voucher_enabled?: boolean;
  }) => request<FuelStation>("/fuel/stations", { method: "POST", body: JSON.stringify(body) }),

  /** Correct a row. Only what is sent changes; the rest is written back to itself. */
  updateFuelStation: (
    stationId: string,
    body: Partial<{
      name: string;
      brand: string;
      brand_label: string;
      aimag: string;
      district: string;
      address: string;
      phone: string;
      opening_hours: string;
      lat: number;
      lon: number;
      total_pumps: number;
      active_pumps: number;
      status: string;
      is_voucher_enabled: boolean;
    }>,
  ) =>
    request<FuelStation>(`/fuel/stations/${stationId}`, {
      method: "PATCH",
      body: JSON.stringify(body),
    }),

  /**
   * Remove a forecourt nothing has happened at.
   *
   * Answers 409 once a delivery has been received there: at that point the row
   * is part of the chain a batch travelled, and the way to take it out of
   * service is to close it.
   */
  deleteFuelStation: (stationId: string) =>
    request<void>(`/fuel/stations/${stationId}`, { method: "DELETE" }),

  /** Add a grade to a forecourt, or change its price, vessel size or availability. */
  setStationGrade: (
    stationId: string,
    body: {
      fuel_type: string;
      fuel_label?: string;
      price_mnt?: number;
      tank_capacity_liters?: number;
      status?: string;
    },
  ) =>
    request<StationGrade>(`/fuel/stations/${stationId}/grades`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),

  /** Stop selling a grade. Different from being out of it — that is `status`. */
  deleteStationGrade: (stationId: string, fuelType: string) =>
    request<void>(`/fuel/stations/${stationId}/grades/${encodeURIComponent(fuelType)}`, {
      method: "DELETE",
    }),

  /** Consignments this organisation has declared, newest first. */
  listFuelShipments: () => request<{ shipments: Shipment[]; count: number }>("/fuel/shipments"),

  /** Declare a consignment arriving at a port. */
  createFuelShipment: (body: {
    declaration_no: string;
    border_port: string;
    origin_country: string;
    exporter: string;
    fuel_type: string;
    declared_liters: number;
    declared_tons?: number | null;
    wagons?: number;
    convoy_code?: string;
    note?: string;
  }) =>
    request<Shipment>("/fuel/shipments", { method: "POST", body: JSON.stringify(body) }),

  /**
   * Move a consignment along.
   *
   * Clearing it is what mints its batch — the number every downstream record
   * carries — so this is the request that starts the chain, not the one that
   * declared the fuel.
   */
  advanceFuelShipment: (
    id: string,
    body: {
      status: string;
      lab_status?: string;
      quality_cert_no?: string;
      octane_tested?: number | null;
      sulfur_ppm?: number | null;
    },
  ) =>
    request<{ id: string; status: string; batch_code: string }>(
      `/fuel/shipments/${id}/status`,
      { method: "POST", body: JSON.stringify(body) },
    ),

  /** This organisation's bases, each with its tanks. */
  listFuelDepots: () => request<{ depots: Depot[]; count: number }>("/fuel/depots"),

  /** Register a base. */
  createFuelDepot: (body: {
    name: string;
    brand?: string;
    aimag?: string;
    district?: string;
    address?: string;
    lat?: number | null;
    lon?: number | null;
    has_rail_siding?: boolean;
    rail_station_code?: string;
  }) => request<Depot>("/fuel/depots", { method: "POST", body: JSON.stringify(body) }),

  /**
   * Add a vessel to a base.
   *
   * No opening level, and there is no endpoint that sets one: a tank enters the
   * system empty and fills through receipts. An opening figure typed into a
   * form would be fuel that entered the country through a text box.
   */
  createFuelTank: (
    depotId: string,
    body: { tank_no: string; tank_type?: string; fuel_type: string; capacity_liters: number },
  ) =>
    request<Tank>(`/fuel/depots/${depotId}/tanks`, {
      method: "POST",
      body: JSON.stringify(body),
    }),

  /** Record a gauge reading. Temperature, density, safety — never the level. */
  updateFuelTank: (
    depotId: string,
    tankId: string,
    body: { temperature_c?: number | null; density_kg_m3?: number | null; safety_status?: string },
  ) =>
    request<Tank>(`/fuel/depots/${depotId}/tanks/${tankId}`, {
      method: "PATCH",
      body: JSON.stringify(body),
    }),

  /** Unload a cleared consignment into a tank. */
  receiveIntoFuelDepot: (
    depotId: string,
    body: { tank_id: string; shipment_id?: string; liters: number; manifest_liters?: number | null; note?: string },
  ) =>
    request<DepotReceipt>(`/fuel/depots/${depotId}/receipts`, {
      method: "POST",
      body: JSON.stringify(body),
    }),

  /** A base's intake history. */
  listFuelDepotReceipts: (depotId: string) =>
    request<{ receipts: DepotReceipt[]; count: number }>(`/fuel/depots/${depotId}/receipts`),

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
