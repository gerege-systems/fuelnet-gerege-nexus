/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

/**
 * The rule the first administrator's password is held to.
 *
 * It repeats tenants.MinAdminPasswordLength, which is the authority: the server
 * refuses a shorter one whatever this says. What it buys is the refusal
 * arriving before the form is submitted rather than after.
 */
export const MIN_SETUP_PASSWORD = 10;
