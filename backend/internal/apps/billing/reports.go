/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Billing's reports. Registered from this module, served by the reports app.
 */

package billing

import (
	"context"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// AppID is what a report declares itself as belonging to. A tenant without this
// app installed never sees these in the list, and cannot run them by key.
const AppID = "io.gerege.nexus.billing"

// registerReports is called from New. Reports are per-module by design: this
// file knows what billing_invoices means and the reports app does not, which is
// the whole reason a Report is an interface rather than a row in a table.
func registerReports() {
	nexus.RegisterReport(revenueByMonth{})
	nexus.RegisterReport(invoiceStatus{})
}

// Sharing. Both billing reports opt in to ScopeFull and neither to
// ScopeCounterparty, and the difference is the schema rather than a decision:
//
//   - ScopeFull is the hierarchical case from §3.5(a) — a parent organisation
//     consolidating a subsidiary's revenue. It needs no filter, so any report
//     can honour it.
//   - ScopeCounterparty is the contracted-parties case, and it needs a column
//     saying which counterparty a row belongs to. `billing_invoices` records a
//     contact *name*, not a registration number, and matching organisations by
//     typed-in name is exactly the mistake §3.5 avoids by keying the grant on a
//     registration number. A report that declared the scope and then filtered
//     on nothing would hand over every invoice the agreement never covered.
//
// A module whose rows do carry a counterparty registration number — the
// transport case the proposal describes — declares both and filters on
// p.Counterparty(). See docs/REPORT_SHARING.md.
func (revenueByMonth) Scopes() []string { return []string{nexus.ReportScopeFull} }
func (invoiceStatus) Scopes() []string  { return []string{nexus.ReportScopeFull} }

// ---------------------------------------------------------------- revenue

type revenueByMonth struct{}

func (revenueByMonth) Key() string { return "billing.revenue_by_month" }
func (revenueByMonth) App() string { return AppID }

func (revenueByMonth) Titles() map[string]string {
	return map[string]string{
		"mn": "Орлого сараар",
		"en": "Revenue by month",
		"ru": "Выручка по месяцам",
		"zh": "按月收入",
		"fr": "Chiffre d'affaires par mois",
		"es": "Ingresos por mes",
		"ar": "الإيرادات شهريًا",
	}
}

func (revenueByMonth) Params() []nexus.ParamSpec {
	return []nexus.ParamSpec{
		{
			Key:  "period",
			Kind: nexus.ParamDateRange,
			Titles: map[string]string{
				"mn": "Хугацаа", "en": "Period", "ru": "Период",
				"zh": "期间", "fr": "Période", "es": "Periodo", "ar": "الفترة",
			},
			// A year rather than the engine's default month: a report grouped
			// by month whose default range is thirty days shows one bar.
			DefaultWindow: 365 * 24 * time.Hour,
		},
		{
			Key:  "status",
			Kind: nexus.ParamSelect,
			Titles: map[string]string{
				"mn": "Төлөв", "en": "Status", "ru": "Статус",
				"zh": "状态", "fr": "Statut", "es": "Estado", "ar": "الحالة",
			},
			Options: []nexus.ParamOption{
				{Value: "ALL", Titles: map[string]string{"mn": "Бүгд", "en": "All"}},
				{Value: "PAID", Titles: map[string]string{"mn": "Төлөгдсөн", "en": "Paid"}},
				{Value: "PENDING", Titles: map[string]string{"mn": "Хүлээгдэж буй", "en": "Pending"}},
				{Value: "CANCELLED", Titles: map[string]string{"mn": "Цуцлагдсан", "en": "Cancelled"}},
			},
			Default: "ALL",
		},
	}
}

func (revenueByMonth) Columns() []nexus.ColumnSpec {
	return []nexus.ColumnSpec{
		{
			Key:    "month",
			Kind:   nexus.ColumnMonth,
			Chart:  nexus.ChartCategory,
			Titles: map[string]string{"mn": "Сар", "en": "Month", "ru": "Месяц", "zh": "月份", "fr": "Mois", "es": "Mes", "ar": "الشهر"},
		},
		{
			Key:    "invoices",
			Kind:   nexus.ColumnNumber,
			Total:  true,
			Titles: map[string]string{"mn": "Нэхэмжлэхийн тоо", "en": "Invoices", "ru": "Счетов", "zh": "发票数", "fr": "Factures", "es": "Facturas", "ar": "الفواتير"},
		},
		{
			Key:    "net",
			Kind:   nexus.ColumnMoney,
			Chart:  nexus.ChartValue,
			Total:  true,
			Titles: map[string]string{"mn": "Дүн (НӨАТ-гүй)", "en": "Net amount", "ru": "Сумма без НДС", "zh": "净额", "fr": "Montant HT", "es": "Importe neto", "ar": "المبلغ الصافي"},
		},
		{
			Key:    "vat",
			Kind:   nexus.ColumnMoney,
			Total:  true,
			Titles: map[string]string{"mn": "НӨАТ", "en": "VAT", "ru": "НДС", "zh": "增值税", "fr": "TVA", "es": "IVA", "ar": "ضريبة القيمة المضافة"},
		},
		{
			Key:    "gross",
			Kind:   nexus.ColumnMoney,
			Chart:  nexus.ChartValue,
			Total:  true,
			Titles: map[string]string{"mn": "Нийт дүн", "en": "Gross amount", "ru": "Итого", "zh": "总额", "fr": "Montant TTC", "es": "Importe bruto", "ar": "الإجمالي"},
		},
	}
}

// Run aggregates in the database rather than in Go.
//
// date_trunc groups a year of invoices into twelve rows before they cross the
// wire; the same report written as "select every invoice and sum in a loop"
// works identically on the demo tenant and falls over on a real one.
//
// `WHERE tenant_id = current_setting(...)` is deliberately not written: the
// clause is `tenant_id = $1`, exactly as every other query in this codebase
// writes it, and the row-level policy underneath is the second layer. Passing
// the tenant as a parameter is also what makes a consolidated run work
// unchanged — see nexus.Engine.Run.
func (r revenueByMonth) Run(ctx context.Context, q nexus.Querier, p nexus.Params) (nexus.Result, error) {
	const query = `
		SELECT date_trunc('month', created_at)::date AS month,
		       count(*)                              AS invoices,
		       coalesce(sum(amount), 0)              AS net,
		       coalesce(sum(vat_amount), 0)          AS vat,
		       coalesce(sum(amount + vat_amount), 0) AS gross
		  FROM billing_invoices
		 WHERE tenant_id = $1
		   AND created_at >= $2 AND created_at <= $3
		   AND ($4 = 'ALL' OR status = $4)
		 GROUP BY 1
		 ORDER BY 1`

	status := p.String("status")
	if status == "" {
		status = "ALL"
	}

	rows, err := q.Query(ctx, query,
		nexus.TenantOf(ctx), p.Time("period_from"), p.Time("period_to"), status)
	if err != nil {
		return nexus.Result{}, err
	}

	collected, err := nexus.Collect(rows, func() (map[string]any, error) {
		var month time.Time
		var invoices int64
		var net, vat, gross float64
		if err := rows.Scan(&month, &invoices, &net, &vat, &gross); err != nil {
			return nil, err
		}
		return map[string]any{
			"month": month, "invoices": invoices,
			"net": net, "vat": vat, "gross": gross,
		}, nil
	})
	if err != nil {
		return nexus.Result{}, err
	}
	return nexus.Result{Rows: collected}, nil
}

// ---------------------------------------------------------------- status

type invoiceStatus struct{}

func (invoiceStatus) Key() string { return "billing.invoice_status" }
func (invoiceStatus) App() string { return AppID }

func (invoiceStatus) Titles() map[string]string {
	return map[string]string{
		"mn": "Нэхэмжлэхийн төлөв",
		"en": "Invoice status",
		"ru": "Статусы счетов",
		"zh": "发票状态",
		"fr": "Statut des factures",
		"es": "Estado de facturas",
		"ar": "حالة الفواتير",
	}
}

func (invoiceStatus) Params() []nexus.ParamSpec {
	return []nexus.ParamSpec{
		{
			Key:  "period",
			Kind: nexus.ParamDateRange,
			Titles: map[string]string{
				"mn": "Хугацаа", "en": "Period", "ru": "Период",
				"zh": "期间", "fr": "Période", "es": "Periodo", "ar": "الفترة",
			},
			DefaultWindow: 90 * 24 * time.Hour,
		},
	}
}

func (invoiceStatus) Columns() []nexus.ColumnSpec {
	return []nexus.ColumnSpec{
		{
			Key:    "status",
			Kind:   nexus.ColumnText,
			Chart:  nexus.ChartCategory,
			Titles: map[string]string{"mn": "Төлөв", "en": "Status", "ru": "Статус", "zh": "状态", "fr": "Statut", "es": "Estado", "ar": "الحالة"},
		},
		{
			Key:    "ebarimt_status",
			Kind:   nexus.ColumnText,
			Titles: map[string]string{"mn": "И-Баримт", "en": "E-Barimt", "ru": "E-Barimt", "zh": "电子票据", "fr": "E-Barimt", "es": "E-Barimt", "ar": "الإيصال الإلكتروني"},
		},
		{
			Key:    "invoices",
			Kind:   nexus.ColumnNumber,
			Chart:  nexus.ChartValue,
			Total:  true,
			Titles: map[string]string{"mn": "Тоо", "en": "Count", "ru": "Количество", "zh": "数量", "fr": "Nombre", "es": "Cantidad", "ar": "العدد"},
		},
		{
			Key:    "gross",
			Kind:   nexus.ColumnMoney,
			Total:  true,
			Titles: map[string]string{"mn": "Нийт дүн", "en": "Gross amount", "ru": "Сумма", "zh": "总额", "fr": "Montant TTC", "es": "Importe", "ar": "الإجمالي"},
		},
	}
}

func (invoiceStatus) Run(ctx context.Context, q nexus.Querier, p nexus.Params) (nexus.Result, error) {
	const query = `
		SELECT status, ebarimt_status,
		       count(*)                              AS invoices,
		       coalesce(sum(amount + vat_amount), 0) AS gross
		  FROM billing_invoices
		 WHERE tenant_id = $1
		   AND created_at >= $2 AND created_at <= $3
		 GROUP BY 1, 2
		 ORDER BY 3 DESC`

	rows, err := q.Query(ctx, query,
		nexus.TenantOf(ctx), p.Time("period_from"), p.Time("period_to"))
	if err != nil {
		return nexus.Result{}, err
	}

	collected, err := nexus.Collect(rows, func() (map[string]any, error) {
		var status, ebarimt string
		var invoices int64
		var gross float64
		if err := rows.Scan(&status, &ebarimt, &invoices, &gross); err != nil {
			return nil, err
		}
		return map[string]any{
			"status": status, "ebarimt_status": ebarimt,
			"invoices": invoices, "gross": gross,
		}, nil
	})
	if err != nil {
		return nexus.Result{}, err
	}
	return nexus.Result{Rows: collected}, nil
}
