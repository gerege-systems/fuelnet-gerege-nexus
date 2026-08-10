import { createHash, randomBytes } from "node:crypto";

/**
 * Signing in with Gerege.
 *
 * The console is a public client: it ships no secret, because a secret in a
 * container image that anybody can pull is not a secret. PKCE is what stands in
 * for one — the platform requires it of every client anyway — and the code
 * exchange happens here on the server so the token never reaches a browser.
 */

export const ISSUER = (process.env.SSO_ISSUER || "https://nexus.gerege.mn").replace(/\/$/, "");
export const CLIENT_ID = process.env.CONSOLE_CLIENT_ID || "gerege-developer-console";
export const CONSOLE_ORIGIN = (
  process.env.CONSOLE_ORIGIN || "https://developer.gerege.mn"
).replace(/\/$/, "");

export const REDIRECT_URI = `${CONSOLE_ORIGIN}/auth/callback`;

// openid is what makes this a sign-in at all; profile and email are what the
// registry records as "who submitted this version". Nothing wider is asked for:
// the console has no business reading a publisher's ERP data.
export const SCOPES = "openid profile email";

function base64url(input: Buffer) {
  return input.toString("base64url");
}

export function newVerifier() {
  // 64 bytes → 86 characters, inside RFC 7636's 43–128.
  return base64url(randomBytes(64));
}

export function challengeFor(verifier: string) {
  return base64url(createHash("sha256").update(verifier).digest());
}

export function newState() {
  return base64url(randomBytes(24));
}

export function authorizationURL(state: string, verifier: string) {
  const query = new URLSearchParams({
    response_type: "code",
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    scope: SCOPES,
    state,
    code_challenge: challengeFor(verifier),
    code_challenge_method: "S256",
  });
  return `${ISSUER}/oauth2/auth?${query.toString()}`;
}

/**
 * Exchanges the authorization code for tokens.
 *
 * Only the id_token is kept. The registry verifies it against the platform's
 * JWKS, so the console needs no access token and holds no refresh token — when
 * the identity expires the person signs in again, which for a console somebody
 * opens a few times a week is the right trade against storing a long-lived
 * credential.
 */
export async function exchangeCode(code: string, verifier: string) {
  const res = await fetch(`${ISSUER}/oauth2/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "authorization_code",
      code,
      redirect_uri: REDIRECT_URI,
      client_id: CLIENT_ID,
      code_verifier: verifier,
    }),
    cache: "no-store",
  });

  if (!res.ok) {
    const detail = await res.text();
    throw new Error(`token exchange failed (${res.status}): ${detail.slice(0, 200)}`);
  }

  const tokens = (await res.json()) as { id_token?: string };
  if (!tokens.id_token) {
    throw new Error("the platform returned no id_token; the openid scope may not be granted");
  }
  return tokens.id_token;
}

/**
 * Reads the claims out of an identity token without verifying it.
 *
 * This is for rendering a name in a corner, nothing else. Every decision that
 * matters — may this person publish, may they review — is made by the registry,
 * which verifies the signature against the issuer's JWKS. A console that
 * trusted its own unverified read of a token would be a console you could
 * promote yourself in by editing a cookie.
 */
export function unverifiedClaims(idToken: string): Record<string, unknown> {
  const parts = idToken.split(".");
  if (parts.length !== 3) return {};
  try {
    return JSON.parse(Buffer.from(parts[1], "base64url").toString("utf8")) as Record<string, unknown>;
  } catch {
    return {};
  }
}
