import { describe, expect, test } from "vitest";

import { ada, grace } from "../test/handlers";
import { type Entry, entryOf, redirectFor } from "./gate";
import { safeRedirect } from "./safe-redirect";

const setup: Entry = { kind: "setup" };
const signIn: Entry = { kind: "sign-in" };
const worker: Entry = { kind: "worker" };
const app: Entry = { kind: "app", account: ada };
const portal: Entry = { kind: "portal", account: grace };

describe("redirectFor", () => {
  test.each([
    ["app", setup, "/setup"],
    ["app", signIn, "/sign-in?redirect=%2Fsystem%3Fx%3D1"],
    ["app", worker, "/worker"],
    ["app", app, null],
    ["app", portal, "/portal"],
    ["portal", setup, "/setup"],
    ["portal", signIn, "/sign-in?redirect=%2Fsystem%3Fx%3D1"],
    ["portal", worker, "/worker"],
    ["portal", app, null],
    ["portal", portal, null],
    ["sign-in", setup, "/setup"],
    ["sign-in", signIn, null],
    ["sign-in", worker, "/worker"],
    ["sign-in", app, "/system"],
    ["sign-in", portal, "/system"],
    ["setup", setup, null],
    ["setup", signIn, "/sign-in"],
    ["setup", worker, "/worker"],
    ["setup", app, "/"],
    ["setup", portal, "/portal"],
    ["worker", setup, "/setup"],
    ["worker", signIn, "/sign-in"],
    ["worker", worker, null],
    ["worker", app, "/"],
    ["worker", portal, "/portal"],
  ] as const)("wanting %s at entry %o goes to %s", (want, entry, target) => {
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
    expect(entryOf({ claim: "controller" }, ada)).toEqual({ kind: "app", account: ada });
  });

  test("a Controller with Grace, a Member, enters the portal as Grace", () => {
    expect(entryOf({ claim: "controller" }, grace)).toEqual({ kind: "portal", account: grace });
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
