/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

// The PDF signing rail the documents app absorbed in migration 00058.

import { request, apiBase } from "./client";

export const esignApi = {
  exportEsignDocument: (id: string, integrationId?: string) =>
    request<{ exported: Array<{ integration_name: string; provider: string; url?: string }> }>(
      `/esign/documents/${id}/export`,
      { method: "POST", body: JSON.stringify(integrationId ? { integration_id: integrationId } : {}) }
    ),

  // Reports (io.gerege.nexus.reports)
  //
  // The engine is generic, so the client is too: nothing here names a report.
  // A screen lists what the tenant may run, asks for one report's declaration,
  // and posts parameters back against it.
  getEsignDocuments: () =>
    request<
      Array<{
        id: string;
        title: string;
        file_name: string;
        status: string;
        page_count: number;
        signer_name: string;
        signer_reg_no: string;
        signer_phone: string;
        signed_at: string | null;
        created_at: string;
      }>
    >("/esign/documents"),

  uploadEsignDocument: (data: { title: string; file_name: string; pdf_base64: string }) =>
    request("/esign/documents", { method: "POST", body: JSON.stringify(data) }),

  checkEsignCert: (data: { phone_no: string; civil_id: string; data?: string }) =>
    request<{ is_valid: boolean; given_name: string; surname: string; common_name: string; uid: string }>(
      "/esign/cert/check",
      { method: "POST", body: JSON.stringify(data) }
    ),

  signEsignDocument: (
    id: string,
    data: { phone_no: string; signer_name: string; signer_reg_no: string; signature_image64: string }
  ) => request<{ status: string; document_id: string; signed_at: string }>(`/esign/documents/${id}/sign`, {
    method: "POST",
    body: JSON.stringify(data),
  }),

  getEsignLogs: () =>
    request<
      Array<{
        id: string;
        document_id: string;
        reg_no: string;
        phone_no: string;
        first_name: string;
        last_name: string;
        action: string;
        created_at: string;
      }>
    >("/esign/logs"),

  downloadEsignDocument: async (id: string, variant: "original" | "signed"): Promise<Blob> => {
    const res = await fetch(`${apiBase()}/esign/documents/${id}/download?variant=${variant}`, {
      credentials: "include",
    });
    if (!res.ok) throw new Error("Download failed");
    return res.blob();
  },

  // Email verification.
  //
  // There is no key management here any more: keys belong to the sending
  // service and are administered there. What this platform keeps is the record
  // of what it asked for.
};
