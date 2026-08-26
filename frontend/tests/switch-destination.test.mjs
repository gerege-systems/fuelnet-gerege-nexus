import assert from "node:assert/strict";
import test from "node:test";

import { switchDestination } from "../lib/nav.mjs";

const MENUS = ["/fuel", "/fuel/depots", "/module/documents/inbox"];

test("a switch stays on the screen the new tenant also has", () => {
  for (const path of ["/fuel", "/fuel/depots", "/module/documents/inbox/abc"]) {
    assert.equal(switchDestination(path, MENUS), path);
  }
});

test("a switch leaves a screen the new tenant has no app for", () => {
  assert.equal(switchDestination("/sso-clients", MENUS), "/apps");
  assert.equal(switchDestination("/fuel", []), "/apps");
  // A sibling whose path merely begins with the same characters is not the app.
  assert.equal(switchDestination("/fuel-cards", MENUS), "/apps");
});

test("the shell's own screens are never left: /menus never lists them", () => {
  for (const path of ["/apps", "/settings/apps", "/profile", "/cp/tenants"]) {
    assert.equal(switchDestination(path, []), path);
  }
});
