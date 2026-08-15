/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Inventory's reports.
 */

package inventory

import (
	"context"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// AppID is the module these reports belong to.
const AppID = "io.gerege.nexus.inventory"

func registerReports() {
	nexus.RegisterReport(stockByWarehouse{})
	nexus.RegisterReport(movementSummary{})
}

// warehouseParam is shared by both reports. The dropdown is filled by running
// OptionsQuery under the caller's tenant binding, so a tenant is offered its
// own warehouses and no others — the list is not a place to leak a name.
func warehouseParam() nexus.ParamSpec {
	return nexus.ParamSpec{
		Key:  "warehouse_id",
		Kind: nexus.ParamUUID,
		Titles: map[string]string{
			"mn": "Агуулах", "en": "Warehouse", "ru": "Склад",
			"zh": "仓库", "fr": "Entrepôt", "es": "Almacén", "ar": "المستودع",
		},
		OptionsQuery: `SELECT id, code || ' — ' || name FROM warehouses
		                WHERE tenant_id = $1 ORDER BY code`,
	}
}

// ---------------------------------------------------------- stock on hand

type stockByWarehouse struct{}

func (stockByWarehouse) Key() string { return "inventory.stock_by_warehouse" }
func (stockByWarehouse) App() string { return AppID }

func (stockByWarehouse) Titles() map[string]string {
	return map[string]string{
		"mn": "Үлдэгдэл агуулахаар",
		"en": "Stock on hand by warehouse",
		"ru": "Остатки по складам",
		"zh": "各仓库库存",
		"fr": "Stock par entrepôt",
		"es": "Existencias por almacén",
		"ar": "المخزون حسب المستودع",
	}
}

func (stockByWarehouse) Params() []nexus.ParamSpec {
	return []nexus.ParamSpec{
		warehouseParam(),
		{
			Key:  "hide_empty",
			Kind: nexus.ParamBool,
			Titles: map[string]string{
				"mn": "Тэг үлдэгдлийг нуух", "en": "Hide zero balances",
				"ru": "Скрыть нулевые остатки", "zh": "隐藏零库存",
				"fr": "Masquer les stocks nuls", "es": "Ocultar saldos cero",
				"ar": "إخفاء الأرصدة الصفرية",
			},
			Default: true,
		},
	}
}

func (stockByWarehouse) Columns() []nexus.ColumnSpec {
	return []nexus.ColumnSpec{
		{Key: "warehouse", Kind: nexus.ColumnText, Chart: nexus.ChartCategory,
			Titles: map[string]string{"mn": "Агуулах", "en": "Warehouse", "ru": "Склад", "zh": "仓库", "fr": "Entrepôt", "es": "Almacén", "ar": "المستودع"}},
		{Key: "sku", Kind: nexus.ColumnText,
			Titles: map[string]string{"mn": "Код", "en": "SKU", "ru": "Артикул", "zh": "编码", "fr": "Référence", "es": "SKU", "ar": "الرمز"}},
		{Key: "product", Kind: nexus.ColumnText,
			Titles: map[string]string{"mn": "Бараа", "en": "Product", "ru": "Товар", "zh": "商品", "fr": "Produit", "es": "Producto", "ar": "المنتج"}},
		{Key: "quantity", Kind: nexus.ColumnNumber, Chart: nexus.ChartValue, Total: true,
			Titles: map[string]string{"mn": "Тоо хэмжээ", "en": "Quantity", "ru": "Количество", "zh": "数量", "fr": "Quantité", "es": "Cantidad", "ar": "الكمية"}},
		{Key: "value", Kind: nexus.ColumnMoney, Total: true,
			Titles: map[string]string{"mn": "Үнийн дүн", "en": "Value", "ru": "Стоимость", "zh": "金额", "fr": "Valeur", "es": "Valor", "ar": "القيمة"}},
	}
}

// Run values the stock at the product's current price.
//
// That is a deliberate simplification and it is stated on the screen rather
// than hidden here: this platform keeps no cost price and no valuation history,
// so "value" is what the goods would sell for today, not what they cost. A
// weighted-average cost would need a purchase-price column that does not exist.
func (stockByWarehouse) Run(ctx context.Context, q nexus.Querier, p nexus.Params) (nexus.Result, error) {
	const query = `
		SELECT w.code || ' — ' || w.name AS warehouse,
		       pr.sku, pr.name           AS product,
		       s.quantity,
		       s.quantity * pr.price     AS value
		  FROM stock_levels s
		  JOIN warehouses w ON w.id = s.warehouse_id AND w.tenant_id = s.tenant_id
		  JOIN products  pr ON pr.id = s.product_id  AND pr.tenant_id = s.tenant_id
		 WHERE s.tenant_id = $1
		   AND ($2::uuid IS NULL OR s.warehouse_id = $2::uuid)
		   AND (NOT $3::boolean OR s.quantity <> 0)
		 ORDER BY w.code, pr.sku`

	rows, err := q.Query(ctx, query,
		nexus.TenantOf(ctx), nullableUUID(p.UUID("warehouse_id")), p.Bool("hide_empty"))
	if err != nil {
		return nexus.Result{}, err
	}

	collected, err := nexus.Collect(rows, func() (map[string]any, error) {
		var warehouse, sku, product string
		var quantity, value float64
		if err := rows.Scan(&warehouse, &sku, &product, &quantity, &value); err != nil {
			return nil, err
		}
		return map[string]any{
			"warehouse": warehouse, "sku": sku, "product": product,
			"quantity": quantity, "value": value,
		}, nil
	})
	if err != nil {
		return nexus.Result{}, err
	}
	return nexus.Result{Rows: collected}, nil
}

// ------------------------------------------------------------- movements

type movementSummary struct{}

func (movementSummary) Key() string { return "inventory.movement_summary" }
func (movementSummary) App() string { return AppID }

func (movementSummary) Titles() map[string]string {
	return map[string]string{
		"mn": "Хөдөлгөөний тойм",
		"en": "Stock movement summary",
		"ru": "Сводка движений",
		"zh": "库存变动汇总",
		"fr": "Résumé des mouvements",
		"es": "Resumen de movimientos",
		"ar": "ملخص الحركات",
	}
}

func (movementSummary) Params() []nexus.ParamSpec {
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
		warehouseParam(),
	}
}

func (movementSummary) Columns() []nexus.ColumnSpec {
	return []nexus.ColumnSpec{
		{Key: "warehouse", Kind: nexus.ColumnText, Chart: nexus.ChartCategory,
			Titles: map[string]string{"mn": "Агуулах", "en": "Warehouse", "ru": "Склад", "zh": "仓库", "fr": "Entrepôt", "es": "Almacén", "ar": "المستودع"}},
		{Key: "movements", Kind: nexus.ColumnNumber, Total: true,
			Titles: map[string]string{"mn": "Гүйлгээний тоо", "en": "Movements", "ru": "Движений", "zh": "变动次数", "fr": "Mouvements", "es": "Movimientos", "ar": "الحركات"}},
		{Key: "received", Kind: nexus.ColumnNumber, Chart: nexus.ChartValue, Total: true,
			Titles: map[string]string{"mn": "Орлого", "en": "Received", "ru": "Приход", "zh": "入库", "fr": "Entrées", "es": "Entradas", "ar": "الوارد"}},
		{Key: "issued", Kind: nexus.ColumnNumber, Chart: nexus.ChartValue, Total: true,
			Titles: map[string]string{"mn": "Зарлага", "en": "Issued", "ru": "Расход", "zh": "出库", "fr": "Sorties", "es": "Salidas", "ar": "الصادر"}},
		{Key: "net", Kind: nexus.ColumnNumber, Total: true,
			Titles: map[string]string{"mn": "Цэвэр өөрчлөлт", "en": "Net change", "ru": "Изменение", "zh": "净变动", "fr": "Variation nette", "es": "Cambio neto", "ar": "صافي التغيير"}},
	}
}

// Run reports issues as a positive figure.
//
// Movements are stored as a signed quantity_change, so the raw sum of outgoing
// stock is negative — and a column headed "Issued" showing -400 is read wrong
// by everybody once. abs() at the query, so both directions read as amounts and
// the net column carries the sign.
func (movementSummary) Run(ctx context.Context, q nexus.Querier, p nexus.Params) (nexus.Result, error) {
	const query = `
		SELECT w.code || ' — ' || w.name AS warehouse,
		       count(*)                                                            AS movements,
		       coalesce(sum(m.quantity_change) FILTER (WHERE m.quantity_change > 0), 0)       AS received,
		       coalesce(abs(sum(m.quantity_change) FILTER (WHERE m.quantity_change < 0)), 0)  AS issued,
		       coalesce(sum(m.quantity_change), 0)                                 AS net
		  FROM stock_movements m
		  JOIN warehouses w ON w.id = m.warehouse_id AND w.tenant_id = m.tenant_id
		 WHERE m.tenant_id = $1
		   AND m.created_at >= $2 AND m.created_at <= $3
		   AND ($4::uuid IS NULL OR m.warehouse_id = $4::uuid)
		 GROUP BY 1
		 ORDER BY 1`

	rows, err := q.Query(ctx, query,
		nexus.TenantOf(ctx), p.Time("period_from"), p.Time("period_to"),
		nullableUUID(p.UUID("warehouse_id")))
	if err != nil {
		return nexus.Result{}, err
	}

	collected, err := nexus.Collect(rows, func() (map[string]any, error) {
		var warehouse string
		var movements int64
		var received, issued, net float64
		if err := rows.Scan(&warehouse, &movements, &received, &issued, &net); err != nil {
			return nil, err
		}
		return map[string]any{
			"warehouse": warehouse, "movements": movements,
			"received": received, "issued": issued, "net": net,
		}, nil
	})
	if err != nil {
		return nexus.Result{}, err
	}
	return nexus.Result{Rows: collected}, nil
}

// nullableUUID turns "no warehouse chosen" into SQL NULL, which the queries
// above read as "every warehouse". An empty string would fail the ::uuid cast.
func nullableUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}
