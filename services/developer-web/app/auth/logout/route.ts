import { NextResponse } from "next/server";
import { CONSOLE_ORIGIN } from "@/lib/oidc";
import { clearSession } from "@/lib/session";

/**
 * Ends the console session and nothing else.
 *
 * It deliberately does not end the platform session: somebody closing the
 * publishing console is not saying they want to be signed out of their
 * organisation's Nexus in the next tab.
 */
export async function POST() {
  await clearSession();
  return NextResponse.redirect(`${CONSOLE_ORIGIN}/`, { status: 303 });
}
