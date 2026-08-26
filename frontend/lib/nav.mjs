// Whether a route lives under a menu path. Compared segment by segment, because
// a raw prefix test also matches a sibling whose path merely begins with the
// same characters: "/products-catalog".startsWith("/products") is true, so the
// Products app would claim the other app's routes, highlight its own tile in
// the rail and render its own menu — leaving the sibling unreachable whenever
// both are installed.
/**
 * @param {string} pathname
 * @param {string} path
 */
export function isUnder(pathname, path) {
  return pathname === path || pathname.startsWith(path.endsWith("/") ? path : path + "/");
}

// Screens the shell owns rather than an app. Every tenant has them, so a switch
// never has to leave one, and they are never in /menus — that list is built from
// the apps a tenant has installed.
const SHELL_PATHS = ["/apps", "/settings", "/profile", "/cp"];

/**
 * Where switching tenant lands: the screen being stood on, when the tenant
 * being moved to still has it. The store otherwise — an app the other
 * organisation never installed has no screen here to return to.
 *
 * @param {string} pathname where the switch was made from
 * @param {string[]} menuPaths the new tenant's menu paths
 * @returns {string}
 */
export function switchDestination(pathname, menuPaths) {
  if (SHELL_PATHS.some((path) => isUnder(pathname, path))) return pathname;
  return menuPaths.some((path) => path && isUnder(pathname, path)) ? pathname : "/apps";
}
