// Public product pages must stay in one list. Layout uses this decision before
// asking for a tenant session; the test suite imports the same function so a
// newly linked marketing page cannot silently become an authenticated screen.
const PUBLIC_ROUTES = new Set([
  "/",
  // The citizen's fuel map. Public for the same reason the product pages are,
  // and for one more: a driver looking for petrol has no organisation to be a
  // member of. The API behind it is on its own public list, in
  // backend/pkg/platform/route_policy_test.go.
  "/map",
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
