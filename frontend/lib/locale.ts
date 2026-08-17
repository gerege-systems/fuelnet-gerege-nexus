/**
 * Where the reader's language is written down, and what it is when nobody has
 * chosen yet.
 *
 * A file of its own, and deliberately *not* `lib/i18n`, because the server
 * needs both: `app/layout.tsx` reads the cookie to title the tab in the right
 * language. `lib/i18n/index.tsx` carries `"use client"`, and a server component
 * importing a value from a client module does not get the value — it gets a
 * client reference, a function standing in for something that will exist in the
 * browser. Nothing warns about it: the cookie was looked up under a function,
 * found nothing, and every tab was titled in Mongolian while its page read in
 * English.
 */
export type Locale = "mn" | "ar" | "zh" | "en" | "fr" | "ru" | "es";

/** The language nobody has chosen yet. */
export const DEFAULT_LOCALE: Locale = "mn";

/**
 * One name for two stores. The switcher writes localStorage (the browser's own
 * memory of the choice) and a cookie (the only half the server can read), and
 * they are the same setting — a reader whose tab disagrees with their page has
 * been told two things about which language they are in.
 */
export const LOCALE_KEY = "locale";
