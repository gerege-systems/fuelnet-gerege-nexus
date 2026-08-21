/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// The organisation app: departments and the people in them.

import { request } from "./client";

export const organisationApi = {
  getDepartments: () =>
    request<Array<{
      id: string; code: string; name: string; parent_id?: string;
      manager_membership_id?: string; manager_name?: string;
      active: boolean; people_count: number;
      tenant_id: string; tenant_name: string;
    }>>("/organisation/departments"),

  createDepartment: (body: { code: string; name: string; parent_id?: string; manager_membership_id?: string }) =>
    request<{ id: string }>("/organisation/departments", { method: "POST", body: JSON.stringify(body) }),

  updateDepartment: (id: string, body: { name: string; parent_id?: string; manager_membership_id?: string }) =>
    request(`/organisation/departments/${id}`, { method: "PUT", body: JSON.stringify(body) }),

  // Archiving keeps the row, because people and documents point at it.
  archiveDepartment: (id: string) =>
    request(`/organisation/departments/${id}/archive`, { method: "POST" }),

  // Deleting removes it, and the server refuses the moment anything does point
  // at it — this is for the unit created by mistake, not the one that was used.
  deleteDepartment: (id: string) => request(`/organisation/departments/${id}`, { method: "DELETE" }),

  // The other half of archiving. It is reversible by design, so the screen that
  // lists what it archived can put one back.
  restoreDepartment: (id: string) =>
    request(`/organisation/departments/${id}/restore`, { method: "POST" }),

  getPeople: () =>
    request<Array<{
      membership_id: string; user_id: string; name: string; email: string;
      phone: string; job_title: string; department_id?: string;
      department_name?: string; active: boolean; is_admin: boolean;
      roles: string[]; joined_at: string;
      // Which organisation this membership is in. The list spans every
      // organisation the session is reading across, so a row can belong to one
      // other than the one being acted in.
      tenant_id: string; tenant_name: string;
    }>>("/organisation/people"),

  updatePerson: (id: string, body: { job_title?: string; department_id?: string }) =>
    request(`/organisation/people/${id}`, { method: "PUT", body: JSON.stringify(body) }),

  setPersonActive: (id: string, active: boolean) =>
    request(`/organisation/people/${id}/${active ? "reactivate" : "deactivate"}`, { method: "POST" }),

};
