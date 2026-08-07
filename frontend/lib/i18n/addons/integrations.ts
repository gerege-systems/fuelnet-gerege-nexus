/**
 * integrations — Connections and webhooks to external systems.
 */
export const integrations = {
  "integrations.view.title": { mn: "Гадаад системийн интеграц ба Webhook", en: "External System Integrations & Webhooks" },
  "integrations.view.create_title": { mn: "Интеграц холбогч бүртгэх", en: "Register Integration Connector" },

  "integrations.field.type": { mn: "Интеграцын төрөл", en: "Integration Type" },
  "integrations.field.name_placeholder": { mn: "жишээ: Борлуулалтын Webhook", en: "e.g. Sales Webhook" },
  "integrations.field.secret": { mn: "Нууц түлхүүр (гарын үсэг)", en: "Secret Key (Signing)" },
  "integrations.field.secret_placeholder": { mn: "HMAC нууц түлхүүр (заавал бус)", en: "Optional HMAC secret" },

  "integrations.type.webhook": { mn: "Webhook хүлээн авагч", en: "Webhook Listener" },
  "integrations.type.government_gateway": { mn: "Төрийн гарц", en: "Government Gateway" },
  "integrations.type.payment_gateway": { mn: "Төлбөрийн гарц", en: "Payment Gateway" },
  "integrations.type.custom_rest": { mn: "Захиалгат REST хаяг", en: "Custom REST Endpoint" },

  "integrations.action.create": { mn: "Интеграц нэмэх", en: "Add Integration" },

  "integrations.message.loading": { mn: "Интеграцуудыг ачаалж байна...", en: "Loading integrations..." },

  "integrations.view.subtitle": { mn: "Төрийн гарц, REST API холбогч, event webhook-уудыг удирдана", en: "Manage government gateways, REST API connectors, and event webhooks" },

  "integrations.field.name": { mn: "Интеграцын нэр", en: "Integration Name" },
  "integrations.field.target_url": { mn: "Хүлээн авах хаяг (URL)", en: "Target Endpoint URL" },

  "integrations.action.register": { mn: "Холбогч бүртгэх", en: "Register Connector" },

  "integrations.message.register_failed": { mn: "Интеграц бүртгэж чадсангүй", en: "Failed to register integration" },
} as const;
