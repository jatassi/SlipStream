import { describe, expect, test } from "vitest";

import { ada } from "../test/handlers";
import { type Entry, entryOf, redirectFor } from "./gate";
import { safeRedirect } from "./safe-redirect";

const setup: Entry = { kind: "setup" };
const signIn: Entry = { kind: "sign-in" };
const worker: Entry = { kind: "worker" };
const app: Entry = { kind: "app", account: ada };

describe("redirectFor", () => {
  test.each([
    ["app", setup, "/setup"],
    ["app", signIn, "/sign-in?redirect=%2Fsystem%3Fx%3D1"],
    ["app", worker, "/worker"],
    ["app", app, null],
    ["sign-in", setup, "/setup"],
    ["sign-in", signIn, null],
    ["sign-in", worker, "/worker"],
    ["sign-in", app, "/system"],
    ["setup", setup, null],
    ["setup", signIn, "/sign-in"],
    ["setup", worker, "/worker"],
    ["setup", app, "/"],
    ["worker", setup, "/setup"],
    ["worker", signIn, "/sign-in"],
    ["worker", worker, null],
    ["worker", app, "/"],
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
