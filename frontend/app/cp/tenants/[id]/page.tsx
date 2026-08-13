"use client";

/**
 * One organisation, as the console may know it: metadata, apps, people, and the
 * two trails — what happened inside the organisation, and what operators did to
 * it. Nothing the organisation keeps on the platform appears here; reading that
 * is impersonation, and impersonation arrives in CP-2 with consent and a
 * reason attached to it.
 */

import React, { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { ArrowLeft, Building2 } from "lucide-react";

import Console from "@/components/cp/Console";
import { formatMoment } from "@/app/cp/page";
import { cp, type TenantDetail } from "@/lib/cp";
import { useI18n } from "@/lib/i18n";

export default function ControlPlaneTenantPage() {
  return (
    <Console>
      <Detail />
    </Console>
  );
}

function Detail() {
  const { t, locale } = useI18n();
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";

  const [tenant, setTenant] = useState<TenantDetail | null>(null);
  const [failure, setFailure] = useState("");

  const load = useCallback(async () => {
    try {
      setTenant(await cp.tenant(id));
      setFailure("");
    } catch (error) {
      setFailure(error instanceof Error ? error.message : String(error));
    }
  }, [id]);

  useEffect(() => {
    if (id) void load();
  }, [id, load]);

  if (failure) {
    return (
      <div className="space-y-4">
        <BackLink label={t("cp.action.back")} />
        <p className="text-sm rounded-lg bg-red-50 text-red-700 border border-red-200 px-3 py-2">
          {t("cp.message.load_failed")}
        </p>
      </div>
    );
  }
  if (!tenant) return <div className="text-slate-500">…</div>;

  return (
    <div className="space-y-6">
      <BackLink label={t("cp.action.back")} />

      <div className="flex items-start gap-3">
        <Building2 className="w-6 h-6 text-slate-400 mt-1" />
        <div>
          <h1 className="text-2xl font-semibold text-slate-900">{tenant.name}</h1>
          <p className="text-sm text-slate-500">
            {tenant.slug}
            {tenant.legal_name ? ` · ${tenant.legal_name}` : ""}
          </p>
        </div>
      </div>

      <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Fact label={t("cp.field.registration")} value={tenant.registration_number || "—"} />
        <Fact label={t("cp.field.tax_number")} value={tenant.tax_number || "—"} />
        <Fact label={t("cp.field.created")} value={formatMoment(tenant.created_at, locale)} />
        <Fact
          label={t("cp.field.last_activity")}
          value={formatMoment(tenant.last_activity_at, locale) || t("cp.message.never")}
        />
      </dl>

      <Card title={t("cp.section.apps")}>
        <Table
          head={[t("cp.field.apps"), t("cp.field.version"), t("cp.field.status"), t("cp.field.installed")]}
          rows={tenant.apps.map((app) => [
            app.name,
            app.version,
            app.enabled ? app.status : `${app.status} · off`,
            formatMoment(app.installed_at, locale),
          ])}
          empty={t("cp.message.no_activity")}
        />
      </Card>

      <Card title={t("cp.section.members")}>
        <Table
          head={[t("cp.field.email"), t("cp.field.organisation"), t("cp.field.roles")]}
          rows={tenant.members.map((member) => [
            member.email,
            member.name,
            member.roles.length ? member.roles.join(", ") : "—",
          ])}
          empty={t("cp.message.no_activity")}
        />
      </Card>

      <Card title={t("cp.section.activity")}>
        <Table
          head={[t("cp.field.when"), t("cp.field.action"), t("cp.field.resource")]}
          rows={tenant.activity.map((entry) => [
            formatMoment(entry.created_at, locale),
            entry.action,
            entry.resource,
          ])}
          empty={t("cp.message.no_activity")}
        />
      </Card>

      <Card title={t("cp.section.operator_actions")}>
        <Table
          head={[t("cp.field.when"), t("cp.field.operator"), t("cp.field.action"), t("cp.field.reason")]}
          rows={tenant.operator_actions.map((entry) => [
            formatMoment(entry.created_at, locale),
            entry.operator_email,
            entry.action,
            entry.reason || "—",
          ])}
          empty={t("cp.message.no_activity")}
        />
      </Card>
    </div>
  );
}

function BackLink({ label }: { label: string }) {
  return (
    <Link href="/cp" className="inline-flex items-center gap-1.5 text-sm text-slate-600 hover:text-slate-900">
      <ArrowLeft className="w-4 h-4" />
      {label}
    </Link>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-white rounded-xl border border-slate-200 shadow-sm px-4 py-3">
      <dt className="text-xs uppercase tracking-wide text-slate-400">{label}</dt>
      <dd className="mt-1 text-sm text-slate-900">{value}</dd>
    </div>
  );
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
      <h2 className="px-4 py-3 border-b border-slate-100 font-medium text-slate-900">{title}</h2>
      {children}
    </section>
  );
}

function Table({ head, rows, empty }: { head: string[]; rows: string[][]; empty: string }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead className="bg-slate-50 text-slate-600">
          <tr>
            {head.map((cell) => (
              <th key={cell} className="text-left font-medium px-4 py-2.5">
                {cell}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {rows.map((row, index) => (
            <tr key={index} className="hover:bg-slate-50">
              {row.map((cell, cellIndex) => (
                <td key={cellIndex} className="px-4 py-2.5 text-slate-700">
                  {cell}
                </td>
              ))}
            </tr>
          ))}
          {rows.length === 0 && (
            <tr>
              <td colSpan={head.length} className="px-4 py-8 text-center text-slate-500">
                {empty}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
