import { describe, expect, test } from "vitest";

import type { FirstrunFirstRun } from "../api/gen/model";
import { ada, grace, noPasskeys } from "../test/handlers";
import { type Entry, entryOf, redirectFor } from "./gate";
import { safeRedirect } from "./safe-redirect";

const setup: Entry = { kind: "setup" };
const signIn: Entry = { kind: "sign-in" };
const worker: Entry = { kind: "worker" };
const newPin: Entry = { kind: "new-pin", account: grace };
const app: Entry = { kind: "app", account: ada };
const portal: Entry = { kind: "portal", account: grace };
const unreachable: Entry = { kind: "unreachable" };
const firstRun: Entry = { kind: "first-run", account: ada, at: "roots" };

const cells = [
  ["app", setup, "/setup"],
  ["app", signIn, "/sign-in?redirect=%2Fsystem%3Fx%3D1"],
  ["app", worker, "/worker"],
  ["app", newPin, "/new-pin"],
  ["app", app, null],
  ["app", portal, "/portal"],
  ["app", unreachable, "/serving"],
  ["app", firstRun, "/setup/roots"],
  ["portal", setup, "/setup"],
  ["portal", signIn, "/sign-in?redirect=%2Fsystem%3Fx%3D1"],
  ["portal", worker, "/worker"],
  ["portal", newPin, "/new-pin"],
  ["portal", app, null],
  ["portal", portal, null],
  ["portal", unreachable, "/serving"],
  ["portal", firstRun, "/setup/roots"],
  ["sign-in", setup, "/setup"],
  ["sign-in", signIn, null],
  ["sign-in", worker, "/worker"],
  ["sign-in", newPin, "/new-pin"],
  ["sign-in", app, "/system"],
  ["sign-in", portal, "/system"],
  ["sign-in", unreachable, "/serving"],
  ["sign-in", firstRun, "/setup/roots"],
  ["setup", setup, null],
  ["setup", signIn, "/sign-in"],
  ["setup", worker, "/worker"],
  ["setup", newPin, "/new-pin"],
  ["setup", app, "/"],
  ["setup", portal, "/portal"],
  ["setup", unreachable, "/serving"],
  ["setup", firstRun, "/setup/roots"],
  ["worker", setup, "/setup"],
  ["worker", signIn, "/sign-in"],
  ["worker", worker, null],
  ["worker", newPin, "/new-pin"],
  ["worker", app, "/"],
  ["worker", portal, "/portal"],
  ["worker", unreachable, "/serving"],
  ["worker", firstRun, "/setup/roots"],
  ["new-pin", setup, "/setup"],
  ["new-pin", signIn, "/sign-in"],
  ["new-pin", worker, "/worker"],
  ["new-pin", newPin, null],
  ["new-pin", app, "/"],
  ["new-pin", portal, "/portal"],
  ["new-pin", unreachable, "/serving"],
  ["new-pin", firstRun, "/setup/roots"],
  ["link", setup, null],
  ["link", signIn, null],
  ["link", worker, null],
  ["link", newPin, null],
  ["link", app, null],
  ["link", portal, null],
  ["link", unreachable, "/serving"],
  ["link", firstRun, null],
  ["first-run", setup, "/setup"],
  ["first-run", signIn, "/sign-in?redirect=%2Fsystem%3Fx%3D1"],
  ["first-run", worker, "/worker"],
  ["first-run", newPin, "/new-pin"],
  ["first-run", app, "/"],
  ["first-run", portal, "/portal"],
  ["first-run", unreachable, "/serving"],
  ["first-run", firstRun, null],
  ["unreachable", setup, "/setup"],
  ["unreachable", signIn, "/sign-in"],
  ["unreachable", worker, "/worker"],
  ["unreachable", newPin, "/new-pin"],
  ["unreachable", app, "/"],
  ["unreachable", portal, "/portal"],
  ["unreachable", unreachable, null],
  ["unreachable", firstRun, "/setup/roots"],
] as const;

describe("redirectFor", () => {
  test.each(cells)("wanting %s at entry %o goes to %s", (want, entry, target) => {
    expect(redirectFor(entry, want, "/system?x=1", "/system")).toBe(target);
  });

  test("a signed-in person on sign-in with no redirect goes home", () => {
    expect(redirectFor(app, "sign-in", "/sign-in")).toBe("/");
  });
});

const controller = { claim: "controller", passkeys: noPasskeys } as const;
const session = (account: typeof ada, pinRequired = false) =>
  ({ account, pinRequired, keyPrompt: "ask" }) as const;
const settingUp = (account: typeof ada) => ({ ...session(account), firstRun: "open" }) as const;
const steps: FirstrunFirstRun = {
  open: true,
  steps: [
    { id: "address", state: "done" },
    { id: "passkey", state: "todo" },
    { id: "roots", state: "todo" },
  ],
};

describe("entryOf", () => {
  test("an Unclaimed Node sets up", () => {
    expect(entryOf({ claim: "unclaimed", passkeys: noPasskeys }, null)).toEqual({ kind: "setup" });
  });

  test("a Controller with no session signs in", () => {
    expect(entryOf(controller, null)).toEqual({
      kind: "sign-in",
    });
  });

  test("a Controller with Ada enters the app as Ada", () => {
    expect(entryOf(controller, session(ada))).toEqual({
      kind: "app",
      account: ada,
    });
  });

  test("a Controller with Grace, a Member, enters the portal as Grace", () => {
    expect(entryOf(controller, session(grace))).toEqual({
      kind: "portal",
      account: grace,
    });
  });

  test("a session that must set a new PIN enters only Set a new PIN, Admin or not", () => {
    expect(entryOf(controller, session(ada, true))).toEqual({
      kind: "new-pin",
      account: ada,
    });
  });

  test("a Worker is managed elsewhere", () => {
    expect(entryOf({ claim: "worker", passkeys: noPasskeys }, null)).toEqual({ kind: "worker" });
  });
});

describe("entryOf in first setup", () => {
  test("an Admin whose first setup is open stands on its first step to do", () => {
    expect(entryOf(controller, settingUp(ada), steps)).toEqual({
      kind: "first-run",
      account: ada,
      at: "roots",
    });
  });

  test("an Admin whose first setup has every step set or skipped stands on the done screen", () => {
    const done: FirstrunFirstRun = { open: true, steps: [{ id: "address", state: "skipped" }] };
    expect(entryOf(controller, settingUp(ada), done)).toEqual({
      kind: "first-run",
      account: ada,
      at: "done",
    });
  });

  test("an Admin enters the app once first setup reads finished", () => {
    expect(entryOf(controller, settingUp(ada), { ...steps, open: false })).toEqual({
      kind: "app",
      account: ada,
    });
    expect(entryOf(controller, { ...session(ada), firstRun: "finished" }, steps)).toEqual({
      kind: "app",
      account: ada,
    });
  });

  test("a new PIN comes before first setup", () => {
    expect(entryOf(controller, { ...settingUp(ada), pinRequired: true }, steps)).toEqual({
      kind: "new-pin",
      account: ada,
    });
  });

  test("a Member never stands in first setup", () => {
    expect(entryOf(controller, settingUp(grace), steps)).toEqual({
      kind: "portal",
      account: grace,
    });
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
