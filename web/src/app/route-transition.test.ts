import { expect, test } from "vitest";

import { transitionTypeFor } from "./route-transition";

test.each([
  ["PUSH", "push"],
  ["FORWARD", "push"],
  ["BACK", "pop"],
  ["REPLACE", null],
] as const)("%s gives %s", (action, type) => {
  expect(transitionTypeFor(action)).toBe(type);
});

test("a Back the browser already animated gives none", () => {
  const popstate = new PopStateEvent("popstate", { hasUAVisualTransition: true });
  expect(transitionTypeFor("BACK", popstate)).toBeNull();
});
