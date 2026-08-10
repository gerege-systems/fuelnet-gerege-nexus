import { NextResponse, type NextRequest } from "next/server";
import { exchangeCode, CONSOLE_ORIGIN } from "@/lib/oidc";
import { takeLoginState, writeSession } from "@/lib/session";

/**
 * Where the platform sends the browser back with an authorization code.
 *
 * The code is exchanged here, server side. What comes back is kept in an
 * httpOnly cookie and the browser is redirected to a plain URL — the code and
 * the token never appear in a page, in history, or in a referrer header.
 */
export async function GET(request: NextRequest) {
  const query = request.nextUrl.searchParams;
  const { state, verifier } = await takeLoginState();

  const failure = (reason: string) =>
    NextResponse.redirect(`${CONSOLE_ORIGIN}/?error=${encodeURIComponent(reason)}`);

  if (query.get("error")) {
    // The platform refused. access_denied here usually means exactly what it
    // says: this person's organisation is not entitled to this application.
    return failure(query.get("error_description") || query.get("error") || "sign-in refused");
  }
  // The state check is the whole defence against somebody else's authorization
  // code being planted in this browser, so a missing one is a failure rather
  // than something to be lenient about.
  if (!state || !verifier || query.get("state") !== state) {
    return failure("the sign-in could not be matched to this browser; please try again");
  }

  const code = query.get("code");
  if (!code) return failure("no authorization code was returned");

  try {
    await writeSession(await exchangeCode(code, verifier));
  } catch (cause) {
    return failure((cause as Error).message);
  }
  return NextResponse.redirect(`${CONSOLE_ORIGIN}/`);
}
