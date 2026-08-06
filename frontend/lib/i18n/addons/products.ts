/**
 * products — Product catalogue: SKUs, names and pricing.
 */
export const products = {
  "products.view.title": { mn: "Барааны каталог", en: "Product Catalog" },
  "products.view.subtitle": { mn: "SKU, барааны нэр, үнийн удирдлага", en: "Manage SKUs, product names, and pricing" },
  "products.view.create_title": { mn: "Шинэ бараа үүсгэх", en: "Create New Product" },

  "products.field.name": { mn: "Барааны нэр", en: "Product Name" },
  "products.field.sku": { mn: "SKU код", en: "SKU" },
  "products.field.sku_placeholder": { mn: "жишээ: PROD-001", en: "e.g. PROD-001" },
  "products.field.price": { mn: "Нэгж үнэ", en: "Unit Price" },

  "products.action.create": { mn: "Шинэ бараа", en: "New Product" },

  "products.message.loading": { mn: "Бараануудыг ачаалж байна...", en: "Loading products..." },
} as const;
