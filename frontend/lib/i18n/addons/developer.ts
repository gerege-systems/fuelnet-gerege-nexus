/**
 * developer — OAuth2 / OIDC client registration for third parties.
 */
export const developer = {
  "developer.view.title": { mn: "Хөгжүүлэгчийн аппууд", en: "Developer apps" },
  "developer.view.subtitle": { mn: "Гуравдагч системд зориулсан OAuth2 / OIDC client бүртгэл", en: "OAuth2 / OIDC client registration for third-party systems" },
  "developer.view.create_title": { mn: "Шинэ OAuth2 client апп бүртгэх", en: "Register New OAuth2 Client App" },

  "developer.field.discovery_endpoint": { mn: "OIDC Discovery хаяг", en: "OIDC Discovery Endpoint" },
  "developer.field.redirect_uris": { mn: "Redirect URI-ууд", en: "Redirect URIs" },
  "developer.field.scopes": { mn: "Scope-ууд", en: "Scopes" },

  "developer.action.create": { mn: "OAuth2 client бүртгэх", en: "Register OAuth2 Client" },

  "developer.message.loading": { mn: "OAuth2 client аппуудыг ачаалж байна...", en: "Loading OAuth2 client apps..." },
  "developer.message.secret_hidden": { mn: "үүсгэх үед нэг удаа харагдана", en: "shown once, at creation" },

  "developer.view.provider_active": { mn: "SSO үйлчилгээ идэвхтэй", en: "Active SSO Provider" },

  "developer.field.application_name": { mn: "Аппликейшны нэр", en: "Application Name" },
  // Client ID and Client Secret are the OAuth2 parameter names a developer
  // copies verbatim, so they stay Latin in every locale — translating the label
  // away from the thing it names would be a worse screen, not a better one.
  "developer.field.client_id": { mn: "Client ID", en: "Client ID" },
  "developer.field.client_secret": { mn: "Client Secret", en: "Client Secret" },
  "developer.field.redirect_uris_hint": { mn: "Redirect URI-ууд (таслалаар тусгаарлана)", en: "Redirect URIs (comma-separated)" },

  "developer.action.register": { mn: "Апп бүртгэх", en: "Register App" },

  "developer.message.create_failed": { mn: "OAuth2 апп үүсгэж чадсангүй", en: "Failed to create OAuth2 app" },
} as const;
