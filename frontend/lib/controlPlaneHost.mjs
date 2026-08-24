/**
 * Decide what the frontend may serve on the control-plane hostname.
 *
 * Kept as a small, dependency-free module so the request boundary can be
 * exercised by Node without booting Next. The API path is admitted for local
 * development; production nginx sends it to the backend before Next sees it.
 *
 * @param {string | null | undefined} requestHost
 * @param {string | null | undefined} configuredHost
 * @param {string} pathname
 * @returns {"other-host" | "redirect" | "allow" | "not-found"}
 */
export function controlPlaneHostDecision(requestHost, configuredHost, pathname) {
  const expected = normaliseHost(configuredHost);
  if (!expected || normaliseHost(requestHost) !== expected) return "other-host";

  if (pathname === "/") return "redirect";
  if (ownsControlPlanePath(pathname)) return "allow";
  return "not-found";
}

/** @param {string} pathname */
function ownsControlPlanePath(pathname) {
  return (
    pathname === "/cp" ||
    pathname.startsWith("/cp/") ||
    pathname === "/api/platform/v1" ||
    pathname.startsWith("/api/platform/v1/")
  );
}

/** @param {string | null | undefined} raw */
function normaliseHost(raw) {
  let host = String(raw || "").trim().toLowerCase();
  const colon = host.lastIndexOf(":");
  if (colon > -1 && !host.slice(colon).includes("]")) host = host.slice(0, colon);
  return host.replace(/\.$/, "");
}
