import {Fingerprint, KeyRound, Layers, Network, ShieldCheck, Waypoints} from "lucide-react";

import type {TranslationKey} from "@/lib/i18n";

/**
 * The published documentation, built from the Markdown in this repository by
 * `docs/site` and served from GitHub Pages.
 *
 * It is a full URL rather than a route because the documentation is not part of
 * this application: it ships on its own schedule, from the same repository, and
 * a reader following it is leaving the product.
 */
export const DOCS_URL = "https://gerege-systems.github.io/open-gerege-nexus/";

type Icon = typeof Fingerprint;

/**
 * What the platform does, as four claims. The order is the order a reader
 * meets them: identity first, because nothing else in the list is reachable
 * before someone has signed in.
 */
export const CAPABILITIES: {icon: Icon; title: TranslationKey; body: TranslationKey}[] = [
  {icon: Fingerprint, title: "website.feature.instant_title", body: "website.feature.instant_body"},
  {icon: Network, title: "website.feature.sso_title", body: "website.feature.sso_body"},
  {icon: ShieldCheck, title: "website.feature.passwordless_title", body: "website.feature.passwordless_body"},
  {icon: Waypoints, title: "website.feature.channels_title", body: "website.feature.channels_body"},
];

/** The guarantees behind the identity claim, as a checklist. */
export const TRUST_POINTS: TranslationKey[] = [
  "website.trust.cookie",
  "website.trust.rbac",
  "website.trust.allowlist",
  "website.trust.audit",
];

/**
 * The three parts a sign-in passes through, drawn left to right in the order
 * a request actually travels them.
 */
export const TECHNOLOGY: {icon: Icon; name: string; body: TranslationKey}[] = [
  {icon: Layers, name: "Gerege Nexus", body: "website.tech.erp_body"},
  {icon: Fingerprint, name: "eID Mongolia", body: "website.tech.eid_body"},
  {icon: KeyRound, name: "OIDC / SSO", body: "website.tech.sso_body"},
];
