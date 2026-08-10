import { cookies } from "next/headers";

/**
 * The console's session.
 *
 * It is a back-end-for-front-end: the authorization code is exchanged on the
 * server and the identity token is kept in an httpOnly cookie, so no token ever
 * reaches the browser's JavaScript. That is worth the extra route — a token in
 * localStorage is readable by anything that manages to run a script on the
 * page, and this token is what the registry accepts as "publish an app".
 *
 * It also means the console never calls the platform's token endpoint from the
 * browser, so the platform needs no CORS entry for this origin. One less thing
 * to configure at the other end, and one less origin allowed to talk to the
 * authorization server.
 */

export const SESSION_COOKIE = "gerege_console_session";
export const STATE_COOKIE = "gerege_console_state";
export const VERIFIER_COOKIE = "gerege_console_verifier";

// Sessions last as long as the identity token they hold. The token carries its
// own expiry and the registry checks it, so this is only about not keeping a
// cookie that is already useless.
const SESSION_TTL_SECONDS = 60 * 60;

// Secure in production, plain over http for a local run — a Secure cookie is
// simply not stored by a browser on http://localhost, which would make the
// console impossible to develop against.
const secure = process.env.NODE_ENV === "production";

export async function readSession(): Promise<string | null> {
  const jar = await cookies();
  return jar.get(SESSION_COOKIE)?.value ?? null;
}

export async function writeSession(idToken: string) {
  const jar = await cookies();
  jar.set(SESSION_COOKIE, idToken, {
    httpOnly: true,
    secure,
    // Lax rather than Strict: the sign-in ends in a redirect from the platform
    // back to this origin, and Strict would drop the cookie on exactly that
    // navigation — the one that just established it.
    sameSite: "lax",
    path: "/",
    maxAge: SESSION_TTL_SECONDS,
  });
}

export async function clearSession() {
  const jar = await cookies();
  jar.delete(SESSION_COOKIE);
}

export async function stashLoginState(state: string, verifier: string) {
  const jar = await cookies();
  const options = { httpOnly: true, secure, sameSite: "lax" as const, path: "/", maxAge: 600 };
  jar.set(STATE_COOKIE, state, options);
  jar.set(VERIFIER_COOKIE, verifier, options);
}

export async function takeLoginState() {
  const jar = await cookies();
  const state = jar.get(STATE_COOKIE)?.value ?? null;
  const verifier = jar.get(VERIFIER_COOKIE)?.value ?? null;
  jar.delete(STATE_COOKIE);
  jar.delete(VERIFIER_COOKIE);
  return { state, verifier };
}
