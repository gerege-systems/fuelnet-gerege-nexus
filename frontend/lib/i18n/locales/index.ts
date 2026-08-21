/**
 * Per-language overlays for the platform's own dictionary.
 *
 * Lookup order in `t()` is overlay → entry[locale] → entry.en, so a term is
 * translated the moment it appears here and falls back to English until then.
 * That is what lets a locale be switched on while it is still only partly
 * translated, instead of waiting for every key.
 *
 * These are the core's overlays only. Each locale directory beside this file
 * also holds one file per app, and those reach `t()` through
 * lib/i18n/registry.ts — an app's translations arrive with the app rather than
 * being spliced into a file the whole platform shares.
 */
import type { Locale } from "../../locale";
import { ar } from "./ar/core";
import { zh } from "./zh/core";
import { fr } from "./fr/core";
import { ru } from "./ru/core";
import { es } from "./es/core";

export const overlays: Partial<Record<Locale, Record<string, string>>> = { ar, zh, fr, ru, es };
