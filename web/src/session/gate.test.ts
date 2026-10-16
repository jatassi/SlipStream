import { describe, expect, test } from "vitest";

import { ada, grace } from "../test/handlers";
import { type Entry, entryOf, redirectFor } from "./gate";
import { safeRedirect } from "./safe-redirect";

const setup: Entry = { kind: "setup" };
const signIn: Entry = { kind: "sign-in" };
const worker: Entry = { kind: "worker" };
const newPin: Entry = { kind: "new-pin", account: grace };
const app: Entry = { kind: "app", account: ada };
const portal: Entry = { kind: "portal", account: grace };

const cells = [
  ["app", setup, "/setup"],
  ["app", signIn, "/sign-in?redirect=%2Fsystem%3Fx%3D1"],
  ["app", worker, "/worker"],
  ["app", newPin, "/new-pin"],
  ["app", app, null],
  ["app", portal, "/portal"],
  ["portal", setup, "/setup"],
  ["portal", signIn, "/sign-in?redirect=%2Fsystem%3Fx%3D1"],
  ["portal", worker, "/worker"],
  ["portal", newPin, "/new-pin"],
  ["portal", app, null],
  ["portal", portal, null],
  ["sign-in", setup, "/setup"],
  ["sign-in", signIn, null],
  ["sign-in", worker, "/worker"],
  ["sign-in", newPin, "/new-pin"],
  ["sign-in", app, "/system"],
  ["sign-in", portal, "/system"],
  ["setup", setup, null],
  ["setup", signIn, "/sign-in"],
  ["setup", worker, "/worker"],
  ["setup", newPin, "/new-pin"],
  ["setup", app, "/"],
  ["setup", portal, "/portal"],
  ["worker", setup, "/setup"],
  ["worker", signIn, "/sign-in"],
  ["worker", worker, null],
  ["worker", newPin, "/new-pin"],
  ["worker", app, "/"],
  ["worker", portal, "/portal"],
  ["new-pin", setup, "/setup"],
  ["new-pin", signIn, "/sign-in"],
  ["new-pin", worker, "/worker"],
  ["new-pin", newPin, null],
  ["new-pin", app, "/"],
  ["new-pin", portal, "/portal"],
  ["link", setup, null],
  ["link", signIn, null],
  ["link", worker, null],
  ["link", newPin, null],
  ["link", app, null],
  ["link", portal, null],
] as const;

describe("redirectFor", () => {
  test.each(cells)("wanting %s at entry %o goes to %s", (want, entry, target) => {
    expect(redirectFor(entry, want, "/system?x=1", "/system")).toBe(target);
  });

  test("a signed-in person on sign-in with no redirect goes home", () => {
    expect(redirectFor(app, "sign-in", "/sign-in")).toBe("/");
  });
});

describe("entryOf", () => {
  test("an Unclaimed Node sets up", () => {
    expect(entryOf({ claim: "unclaimed" }, null)).toEqual({ kind: "setup" });
  });

  test("a Controller with no session signs in", () => {
    expect(entryOf({ claim: "controller" }, null)).toEqual({ kind: "sign-in" });
  });

  test("a Controller with Ada enters the app as Ada", () => {
    expect(entryOf({ claim: "controller" }, { account: ada, pinRequired: false })).toEqual({
      kind: "app",
      account: ada,
    });
  });

  test("a Controller with Grace, a Member, enters the portal as Grace", () => {
    expect(entryOf({ claim: "controller" }, { account: grace, pinRequired: false })).toEqual({
      kind: "portal",
      account: grace,
    });
  });

  test("a session that must set a new PIN enters only Set a new PIN, Admin or not", () => {
    expect(entryOf({ claim: "controller" }, { account: ada, pinRequired: true })).toEqual({
      kind: "new-pin",
      account: ada,
    });
  });

  test("a Worker is managed elsewhere", () => {
    expect(entryOf({ claim: "worker" }, null)).toEqual({ kind: "worker" });
  });
});

describe("safeRedirect", () => {
  test.each([
    ["/system", "/system"],
    ["//evil.example", "/"],
    ["/\\evil.example", "/"],
    ["https://evil.example", "/"],
    ["", "/"],
  ])("%s becomes %s", (raw, safe) => {
    expect(safeRedirect(raw)).toBe(safe);
  });
});
