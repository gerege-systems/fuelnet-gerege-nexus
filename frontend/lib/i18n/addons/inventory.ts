/**
 * inventory — Warehouses, stock levels and movements.
 */
export const inventory = {
  "inventory.view.title": { mn: "Агуулах ба нөөцийн үйл ажиллагаа", en: "Inventory & Warehouse Operations" },
  "inventory.view.warehouses": { mn: "Агуулахууд", en: "Warehouses" },
  "inventory.view.stock_levels": { mn: "Одоогийн үлдэгдэл", en: "Live Stock Levels" },
  "inventory.view.movements": { mn: "Хөдөлгөөний түүх", en: "Stock Movements History" },
  "inventory.view.adjustment": { mn: "Үлдэгдлийн тохируулга", en: "Stock Adjustment" },
  "inventory.view.create_warehouse": { mn: "Агуулах үүсгэх", en: "Create Warehouse" },

  "inventory.field.warehouse": { mn: "Агуулах", en: "Warehouse" },
  "inventory.field.product": { mn: "Бараа", en: "Product" },
  "inventory.field.address": { mn: "Хаяг", en: "Address" },
  "inventory.field.change": { mn: "Өөрчлөлт", en: "Change" },
  "inventory.field.available_quantity": { mn: "Боломжит тоо хэмжээ", en: "Available Quantity" },
  "inventory.field.reference": { mn: "Баримт / Шалтгаан", en: "Reference / Reason" },
  "inventory.field.reference_note": { mn: "Тайлбар", en: "Reference Note" },
  "inventory.field.reference_placeholder": { mn: "жишээ: PO-98421 эсвэл тооллогын зөрүү", en: "e.g. PO-98421 or Physical count adjustment" },
  "inventory.field.datetime": { mn: "Огноо, цаг", en: "Date & Time" },

  "inventory.action.create_warehouse": { mn: "Шинэ агуулах", en: "New Warehouse" },
  "inventory.action.adjust": { mn: "Тохируулга хийх", en: "Adjust Stock" },

  "inventory.message.loading": { mn: "Агуулахын мэдээлэл ачаалж байна...", en: "Loading inventory data..." },
} as const;
