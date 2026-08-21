/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// point of sale — the module lives in pos-gerege-nexus now; these screens have not
// followed it yet. See _departed/README.md.

import { request } from "../client";

export const posApi = {
  getPosRegisters: () => request<Array<{id:string;code:string;name:string;warehouse_id?:string;active:boolean}>>("/pos/registers"),
  createPosRegister: (data:{code:string;name:string;warehouse_id?:string}) => request<{id:string;code:string;name:string;warehouse_id?:string;active:boolean}>("/pos/registers",{method:"POST",body:JSON.stringify(data)}),
  getPosShift: (registerId:string) => request<{shift:null|{id:string;register_id:string;register_code:string;opened_at:string;opening_float:number;closed_at?:string;counted_cash?:number;notes:string;report:{sales:number;gross:number;vat:number;cash_taken:number;card_taken:number;expected_cash:number;variance:number}}}>(`/pos/shift?register_id=${encodeURIComponent(registerId)}`),
  openPosShift: (registerId:string,openingFloat:number,notes="") => request<{id:string}>("/pos/shifts",{method:"POST",body:JSON.stringify({register_id:registerId,opening_float:openingFloat,notes})}),
  closePosShift: (id:string,countedCash:number,notes="") => request(`/pos/shifts/${id}/close`,{method:"POST",body:JSON.stringify({counted_cash:countedCash,notes})}),
  createPosSale: (registerId:string,lines:Array<{product_id:string;quantity:number}>,cash:number,card:number) => request<{id:string;receipt_no:string;total:number;vat_amount:number;cash:number;card:number;change_given:number}>("/pos/sales",{method:"POST",body:JSON.stringify({register_id:registerId,lines,cash,card})}),
  getPosSales: (shiftId:string) => request<Array<{id:string;receipt_no:string;seq:number;total:number;vat_amount:number;cash:number;card:number;change_given:number;created_at:string}>>(`/pos/sales?shift_id=${encodeURIComponent(shiftId)}`),

  // Store
  //
  // A manifest carries release notes since the chronicle: one sentence saying
  // what changed in the version being offered, already resolved to the
  // caller's language by the server. It is what turns "an update is available"
  // into something an administrator can decide about.
};
