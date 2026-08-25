// Public product pages must stay in one list. Layout uses this decision before
// asking for a tenant session; the test suite imports the same function so a
// newly linked marketing page cannot silently become an authenticated screen.
const PUBLIC_ROUTES = new Set([
  "/",
  "/supply",
  "/stations",
  "/vouchers",
  "/oversight",
  "/rollout",
  "/login",
  "/setup",
  "/auth/eid/callback",
  "/oauth/consent",
  "/kiosk",
]);

export function isPublicPath(path) {
  return (
    PUBLIC_ROUTES.has(path) ||
    path.startsWith("/line/") ||
    path === "/cp" ||
    path.startsWith("/cp/")
  );
}
