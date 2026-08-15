/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Three reports: what got done, what was late, and how hard the channel worked.
 *
 * Each is a declaration and nothing more — the list screen, the parameter form,
 * the chart, the Excel export, the schedule that mails it out and the audit
 * entry recording that somebody ran it all come from the engine. What is
 * written here is a title in seven languages, a parameter, some columns and one
 * query.
 *
 * All three declare ScopeFull and none declares ScopeCounterparty. That is not
 * an omission: counterparty scope means "the rows that relate to the reader",
 * and the reader would have to be identified in the data by a
 * `counterparty_ref` — a registration number — which nothing in the task schema
 * carries. A link is not a registration number, and treating it as one would
 * quietly widen a grant meant for one organisation into a view of every
 * subordinate. A report that cannot filter by counterparty must not offer to.
 */

package urtuu

import (
	"context"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

func registerReports() {
	nexus.RegisterReport(taskCompletion{})
	nexus.RegisterReport(slaBreaches{})
	nexus.RegisterReport(channelLoad{})
}

// period is the parameter all three share.
func period(window time.Duration) []nexus.ParamSpec {
	return []nexus.ParamSpec{{
		Key:  "period",
		Kind: nexus.ParamDateRange,
		Titles: map[string]string{
			"mn": "Хугацаа", "en": "Period", "ru": "Период",
			"zh": "期间", "fr": "Période", "es": "Periodo", "ar": "الفترة",
		},
		DefaultWindow: window,
	}}
}

// fullOnly is the scope declaration all three make. See the file comment.
func fullOnly() []string { return []string{nexus.ReportScopeFull} }

// ------------------------------------------------------- task completion

type taskCompletion struct{}

func (taskCompletion) Key() string      { return "urtuu.task_completion" }
func (taskCompletion) App() string      { return ID }
func (taskCompletion) Scopes() []string { return fullOnly() }

func (taskCompletion) Titles() map[string]string {
	return map[string]string{
		"mn": "Даалгаврын биелэлт",
		"en": "Task completion",
		"ru": "Исполнение заданий",
		"zh": "任务完成情况",
		"fr": "Exécution des tâches",
		"es": "Cumplimiento de tareas",
		"ar": "إنجاز المهام",
	}
}

func (taskCompletion) Params() []nexus.ParamSpec { return period(90 * 24 * time.Hour) }

func (taskCompletion) Columns() []nexus.ColumnSpec {
	return []nexus.ColumnSpec{
		{Key: "code", Kind: nexus.ColumnText, Chart: nexus.ChartCategory,
			Titles: map[string]string{"mn": "Код", "en": "Code", "ru": "Код", "zh": "代码", "fr": "Code", "es": "Código", "ar": "الرمز"}},
		{Key: "peer", Kind: nexus.ColumnText,
			Titles: map[string]string{"mn": "Платформ", "en": "Installation", "ru": "Платформа", "zh": "平台", "fr": "Installation", "es": "Instalación", "ar": "المنصّة"}},
		{Key: "raised", Kind: nexus.ColumnNumber, Total: true,
			Titles: map[string]string{"mn": "Нийт", "en": "Raised", "ru": "Всего", "zh": "总数", "fr": "Émises", "es": "Emitidas", "ar": "الإجمالي"}},
		{Key: "completed", Kind: nexus.ColumnNumber, Chart: nexus.ChartValue, Total: true,
			Titles: map[string]string{"mn": "Биелсэн", "en": "Completed", "ru": "Исполнено", "zh": "已完成", "fr": "Terminées", "es": "Completadas", "ar": "منجزة"}},
		{Key: "returned", Kind: nexus.ColumnNumber, Total: true,
			Titles: map[string]string{"mn": "Буцаасан", "en": "Returned", "ru": "Возвращено", "zh": "退回", "fr": "Retournées", "es": "Devueltas", "ar": "معادة"}},
		{Key: "rate", Kind: nexus.ColumnPercent,
			Titles: map[string]string{"mn": "Биелэлт", "en": "Completion", "ru": "Доля исполнения", "zh": "完成率", "fr": "Taux", "es": "Tasa", "ar": "النسبة"}},
		{Key: "avg_days", Kind: nexus.ColumnNumber,
			Titles: map[string]string{"mn": "Дундаж хоног", "en": "Average days", "ru": "Среднее, дней", "zh": "平均天数", "fr": "Jours en moyenne", "es": "Días de media", "ar": "متوسط الأيام"}},
	}
}

// Run groups by code and by the installation the work concerns.
//
// "The installation" is whichever end of the link this row is about — where it
// came from for received work, where it went for delegated. Both are the same
// question from the reader's side: who this task was between.
func (taskCompletion) Run(ctx context.Context, q nexus.Querier, p nexus.Params) (nexus.Result, error) {
	const query = `
		SELECT t.code,
		       coalesce(nullif(op.name, ''), nullif(tp.name, ''), '—')       AS peer,
		       count(*)                                                       AS raised,
		       count(*) FILTER (WHERE t.status IN ('COMPLETED', 'CLOSED'))     AS completed,
		       count(*) FILTER (WHERE t.status = 'RETURNED')                   AS returned,
		       -- Days from raising to the last move, over the tasks that
		       -- finished. Averaging the unfinished ones in would make a
		       -- backlog look like fast work.
		       coalesce(avg(EXTRACT(EPOCH FROM (t.updated_at - t.created_at)) / 86400)
		                FILTER (WHERE t.status IN ('COMPLETED', 'CLOSED')), 0) AS avg_days
		  FROM urtuu_tasks t
		  LEFT JOIN urtuu_peers op ON op.id = t.origin_peer_id
		  LEFT JOIN urtuu_peers tp ON tp.id = t.target_peer_id
		 WHERE t.tenant_id = $1 AND t.created_at >= $2 AND t.created_at <= $3
		 GROUP BY 1, 2
		 ORDER BY 3 DESC, 1
		 LIMIT 500`

	rows, err := q.Query(ctx, query,
		nexus.TenantOf(ctx), p.Time("period_from"), p.Time("period_to"))
	if err != nil {
		return nexus.Result{}, err
	}

	collected, err := nexus.Collect(rows, func() (map[string]any, error) {
		var code, peer string
		var raised, completed, returned int64
		var avgDays float64
		if err := rows.Scan(&code, &peer, &raised, &completed, &returned, &avgDays); err != nil {
			return nil, err
		}
		rate := 0.0
		if raised > 0 {
			rate = float64(completed) / float64(raised) * 100
		}
		return map[string]any{
			"code": code, "peer": peer, "raised": raised, "completed": completed,
			"returned": returned, "rate": rate, "avg_days": round1(avgDays),
		}, nil
	})
	if err != nil {
		return nexus.Result{}, err
	}
	return truncated(collected), nil
}

// ------------------------------------------------------------ SLA breaches

type slaBreaches struct{}

func (slaBreaches) Key() string      { return "urtuu.sla_breaches" }
func (slaBreaches) App() string      { return ID }
func (slaBreaches) Scopes() []string { return fullOnly() }

func (slaBreaches) Titles() map[string]string {
	return map[string]string{
		"mn": "Хугацаа хэтрэлт",
		"en": "SLA breaches",
		"ru": "Нарушения сроков",
		"zh": "超期情况",
		"fr": "Dépassements de délai",
		"es": "Plazos incumplidos",
		"ar": "تجاوزات المهلة",
	}
}

func (slaBreaches) Params() []nexus.ParamSpec { return period(90 * 24 * time.Hour) }

func (slaBreaches) Columns() []nexus.ColumnSpec {
	return []nexus.ColumnSpec{
		{Key: "title", Kind: nexus.ColumnText, Chart: nexus.ChartCategory,
			Titles: map[string]string{"mn": "Даалгавар", "en": "Task", "ru": "Задание", "zh": "任务", "fr": "Tâche", "es": "Tarea", "ar": "المهمة"}},
		{Key: "code", Kind: nexus.ColumnText,
			Titles: map[string]string{"mn": "Код", "en": "Code", "ru": "Код", "zh": "代码", "fr": "Code", "es": "Código", "ar": "الرمز"}},
		{Key: "peer", Kind: nexus.ColumnText,
			Titles: map[string]string{"mn": "Платформ", "en": "Installation", "ru": "Платформа", "zh": "平台", "fr": "Installation", "es": "Instalación", "ar": "المنصّة"}},
		{Key: "status", Kind: nexus.ColumnText,
			Titles: map[string]string{"mn": "Төлөв", "en": "Status", "ru": "Состояние", "zh": "状态", "fr": "État", "es": "Estado", "ar": "الحالة"}},
		{Key: "deadline", Kind: nexus.ColumnDate,
			Titles: map[string]string{"mn": "Товлосон", "en": "Due", "ru": "Срок", "zh": "期限", "fr": "Échéance", "es": "Plazo", "ar": "المهلة"}},
		{Key: "days_late", Kind: nexus.ColumnNumber, Chart: nexus.ChartValue,
			Titles: map[string]string{"mn": "Хэтэрсэн хоног", "en": "Days late", "ru": "Дней просрочки", "zh": "超期天数", "fr": "Jours de retard", "es": "Días de retraso", "ar": "أيام التأخير"}},
	}
}

// Run lists what missed its deadline, worst first.
//
// A task that is still open counts from now; one that finished counts from when
// it finished, so a fortnight late and settled is not the same row as a
// fortnight late and still open. Closed tasks are excluded: the originator
// accepting the outcome is what ends the question.
func (slaBreaches) Run(ctx context.Context, q nexus.Querier, p nexus.Params) (nexus.Result, error) {
	const query = `
		SELECT t.title, t.code,
		       coalesce(nullif(op.name, ''), nullif(tp.name, ''), '—') AS peer,
		       t.status, t.deadline::date,
		       EXTRACT(EPOCH FROM (
		           CASE WHEN t.status IN ('COMPLETED', 'RETURNED') THEN t.updated_at
		                ELSE NOW() END - t.deadline)) / 86400          AS days_late
		  FROM urtuu_tasks t
		  LEFT JOIN urtuu_peers op ON op.id = t.origin_peer_id
		  LEFT JOIN urtuu_peers tp ON tp.id = t.target_peer_id
		 WHERE t.tenant_id = $1
		   AND t.deadline IS NOT NULL AND t.status <> 'CLOSED'
		   AND t.created_at >= $2 AND t.created_at <= $3
		   AND (CASE WHEN t.status IN ('COMPLETED', 'RETURNED') THEN t.updated_at
		             ELSE NOW() END) > t.deadline
		 ORDER BY 6 DESC
		 LIMIT 500`

	rows, err := q.Query(ctx, query,
		nexus.TenantOf(ctx), p.Time("period_from"), p.Time("period_to"))
	if err != nil {
		return nexus.Result{}, err
	}

	collected, err := nexus.Collect(rows, func() (map[string]any, error) {
		var title, code, peer, status string
		var deadline time.Time
		var late float64
		if err := rows.Scan(&title, &code, &peer, &status, &deadline, &late); err != nil {
			return nil, err
		}
		return map[string]any{
			"title": title, "code": code, "peer": peer, "status": status,
			"deadline": deadline, "days_late": round1(late),
		}, nil
	})
	if err != nil {
		return nexus.Result{}, err
	}
	return truncated(collected), nil
}

// ------------------------------------------------------------ channel load

type channelLoad struct{}

func (channelLoad) Key() string      { return "urtuu.channel_load" }
func (channelLoad) App() string      { return ID }
func (channelLoad) Scopes() []string { return fullOnly() }

func (channelLoad) Titles() map[string]string {
	return map[string]string{
		"mn": "Сувгийн ачаалал",
		"en": "Channel load",
		"ru": "Нагрузка канала",
		"zh": "通道负载",
		"fr": "Charge du canal",
		"es": "Carga del canal",
		"ar": "حِمل القناة",
	}
}

func (channelLoad) Params() []nexus.ParamSpec { return period(30 * 24 * time.Hour) }

func (channelLoad) Columns() []nexus.ColumnSpec {
	return []nexus.ColumnSpec{
		{Key: "peer", Kind: nexus.ColumnText, Chart: nexus.ChartCategory,
			Titles: map[string]string{"mn": "Холбоос", "en": "Link", "ru": "Связь", "zh": "连接", "fr": "Lien", "es": "Enlace", "ar": "الرابط"}},
		{Key: "envelopes", Kind: nexus.ColumnNumber, Chart: nexus.ChartValue, Total: true,
			Titles: map[string]string{"mn": "Дугтуй", "en": "Envelopes", "ru": "Конвертов", "zh": "信封数", "fr": "Enveloppes", "es": "Sobres", "ar": "المظاريف"}},
		{Key: "delivered", Kind: nexus.ColumnNumber, Total: true,
			Titles: map[string]string{"mn": "Хүргэгдсэн", "en": "Delivered", "ru": "Доставлено", "zh": "已送达", "fr": "Remises", "es": "Entregados", "ar": "مُسلَّمة"}},
		{Key: "pending", Kind: nexus.ColumnNumber, Total: true,
			Titles: map[string]string{"mn": "Хүлээгдэж буй", "en": "Outstanding", "ru": "Ожидают", "zh": "待送达", "fr": "En attente", "es": "Pendientes", "ar": "معلّقة"}},
		{Key: "retries", Kind: nexus.ColumnNumber, Total: true,
			Titles: map[string]string{"mn": "Дахин оролдлого", "en": "Retries", "ru": "Повторов", "zh": "重试次数", "fr": "Reprises", "es": "Reintentos", "ar": "المحاولات"}},
	}
}

// Run counts what actually went over each link.
//
// Retries are `attempts - 1` summed rather than `attempts`: an envelope that
// went first time has one attempt and no retry, and a column that counted it as
// one would make a healthy channel look like a struggling one.
func (channelLoad) Run(ctx context.Context, q nexus.Querier, p nexus.Params) (nexus.Result, error) {
	const query = `
		SELECT coalesce(nullif(p.name, ''), '—')                     AS peer,
		       count(*)                                              AS envelopes,
		       count(*) FILTER (WHERE d.delivered_at IS NOT NULL)     AS delivered,
		       count(*) FILTER (WHERE d.delivered_at IS NULL)         AS pending,
		       coalesce(sum(greatest(d.attempts - 1, 0)), 0)          AS retries
		  FROM urtuu_deliveries d
		  JOIN urtuu_peers p ON p.id = d.peer_id
		 WHERE d.tenant_id = $1 AND d.created_at >= $2 AND d.created_at <= $3
		 GROUP BY 1
		 ORDER BY 2 DESC
		 LIMIT 500`

	rows, err := q.Query(ctx, query,
		nexus.TenantOf(ctx), p.Time("period_from"), p.Time("period_to"))
	if err != nil {
		return nexus.Result{}, err
	}

	collected, err := nexus.Collect(rows, func() (map[string]any, error) {
		var peer string
		var envelopes, delivered, pending, retries int64
		if err := rows.Scan(&peer, &envelopes, &delivered, &pending, &retries); err != nil {
			return nil, err
		}
		return map[string]any{
			"peer": peer, "envelopes": envelopes, "delivered": delivered,
			"pending": pending, "retries": retries,
		}, nil
	})
	if err != nil {
		return nexus.Result{}, err
	}
	return truncated(collected), nil
}

// ----------------------------------------------------------------- helpers

// truncated says so when the limit was reached. A limit that is not said is a
// lie: five hundred rows read as the whole answer.
func truncated(rows []map[string]any) nexus.Result {
	result := nexus.Result{Rows: rows}
	if len(rows) == 500 {
		result.Notes = append(result.Notes, nexus.Note{
			Level:   "warning",
			Message: "Эхний 500 мөрийг харуулав; жагсаалт таслагдсан.",
		})
	}
	return result
}

// round1 keeps a day count readable. Three decimal places of "days late" is
// precision nobody asked for and every export has to carry.
func round1(value float64) float64 {
	return float64(int64(value*10+0.5)) / 10
}
