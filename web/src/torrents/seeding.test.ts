import { expect, test } from "vitest";

import { seedingOf } from "../test/torrents";
import {
  endlessSentence,
  goalSentence,
  marginSentence,
  ruleSentence,
  seedLine,
  type Seeding,
} from "./seeding";

const day = 86_400;
const gb = 1.2 * 1024 ** 3;

const owing = (more: Partial<Seeding> = {}) =>
  seedingOf({
    ratio: { value: 0.4, rule: 1, goal: 2 },
    time: { value: 7 * day, rule: 10 * day + 7200, goal: 14 * day, margin: 7200 },
    state: "owing",
    owes: { seconds: 3 * day + 4 * 3600, bytes: gb },
    clears: "either",
    ...more,
  });

const write = (seeding: Seeding) =>
  [
    seedLine(seeding),
    ruleSentence("seedpool", seeding),
    marginSentence("seedpool", seeding),
    goalSentence(seeding),
    endlessSentence("seedpool", seeding),
  ].filter((words) => words !== undefined && words !== "");

const cases: readonly [string, Seeding, readonly string[]][] = [
  [
    "owes time",
    owing({ owes: { seconds: 3 * day + 4 * 3600, bytes: 0 }, clears: "time" }),
    ["3 days 4 h", "10 days", "2 h", "2.0", "14 days", "whichever comes first", "removes"],
  ],
  ["owes either one", owing(), ["seeding or 1.2 GB", "1.0 or 10 days"]],
  ["owes both", owing({ clears: "both" }), ["seeding and 1.2 GB", "1.0 and 10 days"]],
  [
    "clears on ratio, pausing once met",
    owing({ owes: { seconds: 0, bytes: gb }, clears: "ratio", action: "pause" }),
    ["1.2 GB", "ratio of 1.0", "pauses"],
  ],
  [
    "seeds Forever",
    seedingOf({ state: "met", combine: "forever", endless: "forever", clears: "time" }),
    ["No seeding limit", "Forever"],
  ],
  ["was resumed", seedingOf({ state: "met", action: "pause", endless: "resumed" }), ["resumed"]],
  [
    "has no known rule",
    seedingOf({ state: "owing", clears: "none", endless: "no-rule" }),
    ["seedpool", "Settings > Trackers"],
  ],
  [
    "has no known Tracker",
    seedingOf({ state: "owing", clears: "none", endless: "no-tracker" }),
    ["doesn't know which Tracker"],
  ],
  [
    "has only some files selected",
    seedingOf({ state: "owing", clears: "none", endless: "partial" }),
    ["seedpool", "some of its files"],
  ],
];

test("each seeding state reads its facts in sentences", () => {
  for (const [state, seeding, facts] of cases) {
    const words = write(seeding);
    for (const sentence of words) {
      expect(sentence, state).toMatch(/^\S/u);
    }
    const all = words.join(" ");
    for (const fact of facts) {
      expect(all, state).toContain(fact);
    }
  }
});
