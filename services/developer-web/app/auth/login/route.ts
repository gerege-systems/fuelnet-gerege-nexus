import { NextResponse, type NextRequest } from "next/server";
import { authorizationURL, newState, newVerifier } from "@/lib/oidc";
import { stashLoginState } from "@/lib/session";

/**
 * Starts the sign-in.
 *
 * The verifier and the state are put in short-lived httpOnly cookies rather
 * than in the URL: they are the two things that make the round trip
 * unforgeable, and a value the browser can read is one a script on the page can
 * read too.
 */
export async function GET(_request: NextRequest) {
  const state = newState();
  const verifier = newVerifier();
  await stashLoginState(state, verifier);
  return NextResponse.redirect(authorizationURL(state, verifier));
}
