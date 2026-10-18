import { expect, test } from "vitest";

import type { NodesCodeState, NodesNode } from "../../../../api/gen/model";
import { home, seedbox } from "../../../../test/nodes";
import { type CodeAnswer, type Facts, addNodeState, countdown } from "./state";

const id = "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d";
const now = new Date("2026-10-18T21:00:00Z");
const expiry = "2026-10-18T21:59:30Z";
const minted = { id, code: "SLP1-AAAAA", expiresAt: expiry };

const known = (state: NodesCodeState): CodeAnswer => ({ kind: "known", state });

const facts = (over: Partial<Facts>): Facts => ({
  id,
  answer: known({ state: "open" }),
  minted,
  expiresAt: expiry,
  nodes: [home],
  now,
  ...over,
});

const vps: NodesNode = { ...seedbox, id: "7d9f1b3c-5e7a-4c9e-8b1d-3f5a7c9e1b2d", name: "vps" };

test("an open code this page made shows its text and the time left", () => {
  expect(addNodeState(facts({}))).toEqual({
    kind: "open",
    code: "SLP1-AAAAA",
    left: "59:30",
    dial: { kind: "none" },
  });
});

test("after a reload the code is still followed, but its text is gone", () => {
  expect(addNodeState(facts({ minted: undefined }))).toMatchObject({
    kind: "open",
    code: undefined,
    left: "59:30",
  });
});

test("a code expires on the server's word or once the clock passes its expiry", () => {
  expect(addNodeState(facts({ answer: known({ state: "expired" }) }))).toEqual({
    kind: "expired",
  });
  expect(addNodeState(facts({ now: new Date(expiry) }))).toEqual({ kind: "expired" });
});

test("a code the Controller no longer has was revoked, unless it had expired", () => {
  expect(addNodeState(facts({ answer: { kind: "gone" } }))).toEqual({ kind: "revoked" });
  expect(addNodeState(facts({ answer: { kind: "gone" }, now: new Date(expiry) }))).toEqual({
    kind: "expired",
  });
});

test("a pasted Node address reads dialing, then can't reach once the report time passes", () => {
  const dialing = {
    address: "SLN1-ZZZZZ",
    addresses: ["203.0.113.9:8485"],
    since: "2026-10-18T21:00:00Z",
    unreachable: false,
  };
  expect(addNodeState(facts({ answer: known({ state: "open", dialing }) }))).toMatchObject({
    dial: { kind: "dialing", addresses: ["203.0.113.9:8485"] },
  });
  expect(
    addNodeState(
      facts({ answer: known({ state: "open", dialing: { ...dialing, unreachable: true } }) }),
    ),
  ).toMatchObject({ dial: { kind: "cant-reach", addresses: ["203.0.113.9:8485"] } });
});

test("a used code waits for its Node's row, then names who dials whom", () => {
  const used = known({ state: "used", nodeId: vps.id });
  expect(addNodeState(facts({ answer: used }))).toEqual({ kind: "pairing" });
  expect(
    addNodeState(
      facts({ answer: used, nodes: [home, { ...vps, dial: { by: "node", controllerDials: [] } }] }),
    ),
  ).toMatchObject({ kind: "paired", reached: "vps dials home" });
  expect(
    addNodeState(
      facts({
        answer: used,
        nodes: [
          home,
          { ...vps, dial: { by: "controller", controllerDials: ["203.0.113.9:8485"] } },
        ],
      }),
    ),
  ).toMatchObject({ kind: "paired", reached: "home dials vps" });
  expect(addNodeState(facts({ answer: used, nodes: [home, vps] }))).toMatchObject({
    kind: "paired",
    reached: "Not connected",
  });
});

test("the countdown ticks in minutes and seconds, hours past one, days past a day", () => {
  expect(countdown(59_000)).toBe("0:59");
  expect(countdown(14 * 60_000 + 5000)).toBe("14:05");
  expect(countdown(5 * 3_600_000 + 61_000)).toBe("5:01:01");
  expect(countdown(26 * 3_600_000)).toBe("1 day 2 h");
  expect(countdown(7 * 24 * 3_600_000)).toBe("7 days 0 h");
});
