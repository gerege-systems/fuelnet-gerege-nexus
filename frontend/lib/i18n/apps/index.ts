/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

/**
 * The apps this build carries, registering their own words.
 *
 * Imported for the side effect: each module calls registerDictionary at the top
 * level, so by the time anything renders, every app in this binary has handed
 * its strings over. A distribution's app does the same from its own code and
 * this file never learns its name.
 *
 * Why eager, and not from each app's route.
 *
 * The obvious arrangement is for an app to register in its own
 * `app/<slug>/layout.tsx`, so that its translations are loaded only where they
 * are read. That is wrong here, and measurably so: an app's keys are not only
 * read on the app's own routes.
 *
 *   ai            components/AICopilot.tsx, mounted in the shell header on
 *                 every page (components/Layout.tsx:359, :399)
 *   esign         app/module/documents/* — six screens under another app
 *   urtuu         app/settings/urtuu/page.tsx — a platform settings route
 *   storefront    components/landing/Storefront.tsx — the signed-out page
 *
 * Registering from the route would leave each of those rendering English while
 * the rest of the page is in Mongolian, and nothing would fail: `t()` falls
 * back to English by design, which is right for a term nobody has translated
 * and wrong for one that is sitting in a file that has not been imported. A
 * silent half-translated screen is exactly the failure this split must not
 * introduce.
 *
 * So the list is here rather than nowhere. What it is not is the old
 * arrangement under a new name: index.tsx no longer imports any app, no app
 * key is part of TranslationKey, and a distribution adds an app without
 * touching either this file or that one. Adding an in-repo app is one line
 * here, in a file that exists for that line.
 *
 * Six entries were removed on 2026-08-21 — contacts, products, inventory,
 * billing, gov and appstore_modules. Their screens went with them: the core
 * served no route for any of them, so every one of those pages was calling an
 * endpoint that answered 404.
 */

import "./fuel";
import "./sso_clients";
import "./storefront";
import "./urtuu";
