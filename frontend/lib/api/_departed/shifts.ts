/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// device shifts — the module lives in pos-gerege-nexus now; these screens have not
// followed it yet. See _departed/README.md.

import { request } from "../client";

export const shiftsApi = {
  getCurrentShift: () => request<{shift:null|{id:string;membership_id:string;opened_at:string;opening_float:number} }>("/devices/shifts/current"),
  openShift: (openingFloat:number,notes="") => request<{id:string;opened_at:string}>("/devices/shifts/open",{method:"POST",body:JSON.stringify({opening_float:openingFloat,notes})}),
  closeShift: (closingTotal:number,notes="") => request<{id:string;status:string}>("/devices/shifts/close",{method:"POST",body:JSON.stringify({closing_total:closingTotal,notes})}),

  // Касс — io.gerege.nexus.pos (pos-gerege-nexus). Тэр модульгүй суулгац
  // эдгээрт 403 хариулна, тэр нь "энэ бүтээгдэхүүн касс биш" гэсэн жинхэнэ
  // хариу; app/pos/page.tsx түүнийг уншаад төхөөрөмжийн ээлж рүү шилждэг.
  //
  // Ээлж нь кассын цэгт хамаарна, төхөөрөмжид биш: дээрх /devices/shifts/*
  // нь device токен шаарддаг тул лангуун дээрх хөтөч түүнийг нээж чадахгүй.
};
