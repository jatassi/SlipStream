import { expect, test } from "vitest";

import { ApiError } from "../api/errors";
import type { NodesBoardNode, NodesGesture, NodesRoleView } from "../api/gen/model";
import { board, groups, homeSeat, outcome, seedboxSeat } from "../test/roles";
import { draftOf, hintOf, refusedOutcome, roleLine, withGroup, lockedOf } from "./roles";

const clock = { now: new Date("2026-10-12T22:00:00Z"), zone: "UTC" };
const online = { standing: "in-step", since: "2026-10-12T09:00:00Z" } as const;
const offline = { standing: "offline", since: "2026-10-12T21:31:00Z" } as const;
const torrents = { units: 3, noun: "torrents", bytes: 41 * 1024 ** 3 };

const entry = (role: string) => {
  const found = groups.flatMap(({ roles }) => roles).find((candidate) => candidate.role === role);
  if (found === undefined) {
    throw new Error(role);
  }
  return found;
};

test("each Role's state reads as a line", () => {
  const lapsedSeedbox: NodesBoardNode = {
    ...seedboxSeat,
    roles: [{ role: "searching", state: "lapsed" }],
  };
  const shown = board(homeSeat, lapsedSeedbox);
  const line = (view: NodesRoleView, standing: typeof online | typeof offline = online) =>
    roleLine(shown, view, standing, clock);
  expect(line({ role: "searching", state: "holding" })).toBe("Holding");
  expect(line({ role: "downloading", state: "holding", held: torrents })).toBe(
    "Holding 3 torrents (41 GB)",
  );
  expect(line({ role: "downloading", state: "draining", held: torrents })).toBe(
    "Draining · 3 torrents (41 GB) left",
  );
  expect(line({ role: "downloading", state: "writing-off", held: torrents })).toBe(
    "Writing off 3 torrents (41 GB)",
  );
  expect(line({ role: "downloading", state: "away" }, offline)).toBe("Offline since 21:31");
  expect(line({ role: "downloading", state: "lapsed" }, offline)).toBe(
    "Offline since 21:31 · past the offline grace",
  );
  expect(line({ role: "searching", state: "covering" })).toBe("Covering for seedbox");
});

test("a probe that fails or is weak, or a guardrail on adding the Role, hints why", () => {
  const failing: NodesBoardNode = {
    ...seedboxSeat,
    assigned: [],
    probes: [
      {
        probe: "engine-starts",
        fit: "fails",
        reason: "engine_not_listening",
        words: "Its torrent engine isn't listening for peers.",
        at: "2026-10-12T10:00:00Z",
      },
    ],
  };
  const gestures: NodesGesture[] = [
    { role: "downloading", to: failing.id, verdict: "block", word: "Can't run here" },
    { role: "relaying", to: failing.id, verdict: "block", word: "relay-fra already relays" },
    { role: "searching", to: failing.id, verdict: "allow", word: "Adds Searching" },
  ];
  const gestured = { ...board(homeSeat, failing), gestures };
  expect(hintOf(gestured, failing, entry("downloading"))).toEqual({
    tone: "block",
    text: "Can't run here. Its torrent engine isn't listening for peers.",
  });
  expect(hintOf(gestured, failing, entry("relaying"))).toEqual({
    tone: "block",
    text: "relay-fra already relays",
  });
  expect(hintOf(gestured, failing, entry("searching"))).toBeUndefined();

  const ticked: NodesBoardNode = { ...failing, assigned: ["downloading"] };
  expect(hintOf(gestured, ticked, entry("downloading"))).toEqual({
    tone: "block",
    text: "Its torrent engine isn't listening for peers.",
  });
});

test("the Controller's pinned Roles stay ticked when its group switches off", () => {
  const shown = board(homeSeat, seedboxSeat);
  const locked = lockedOf(shown, homeSeat);
  expect([...locked]).toEqual(["serving"]);
  expect([...lockedOf(shown, seedboxSeat)]).toEqual([]);
  const reach = groups.find(({ group }) => group === "reach");
  if (reach === undefined) {
    throw new Error("no reach group");
  }
  const draft = draftOf(shown, homeSeat);
  const relaying = withGroup(draft, reach, true, locked);
  expect([...withGroup(relaying, reach, false, locked)].toSorted()).toEqual([...draft].toSorted());
  expect(relaying.has("relaying")).toBe(true);
});

test("a refusal's Outcome is read from its whole body", () => {
  const carried = outcome({ basis: "basis-2" });
  const refused = new ApiError(
    { status: 409, code: "roles_stale", title: "The Roles changed" },
    { status: 409, code: "roles_stale", title: "The Roles changed", outcome: carried },
  );
  expect(refusedOutcome(refused)).toEqual(carried);
  expect(
    refusedOutcome(new ApiError({ status: 404, code: "node_not_found", title: "No Node" })),
  ).toBeUndefined();
  expect(refusedOutcome(new Error("nope"))).toBeUndefined();
});
