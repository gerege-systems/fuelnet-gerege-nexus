import assert from "node:assert/strict";
import test from "node:test";

import { isPublicPath } from "../lib/publicPath.mjs";

test("FuelNet product pages do not require a tenant session", () => {
  for (const path of [
    "/",
    "/supply",
    "/stations",
    "/vouchers",
    "/oversight",
    "/rollout",
  ]) {
    assert.equal(isPublicPath(path), true, path);
  }
});

test("authenticated platform pages remain protected", () => {
  for (const path of [
    "/apps",
    "/profile",
    "/settings",
    "/module/documents",
    "/supply/private",
  ]) {
    assert.equal(isPublicPath(path), false, path);
  }
});
