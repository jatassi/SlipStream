import { expect, test } from "vitest";

import {
  atWords,
  countWords,
  durationWords,
  inWords,
  knownZone,
  lengthWords,
  sinceWords,
} from "./format";

const local = (day: number, hours: number, minutes = 0) =>
  new Date(2026, 9, day, hours, minutes).toISOString();

const at = (day: number, hours: number, minutes = 0) => new Date(2026, 9, day, hours, minutes);

const browser = (now: Date) => ({ now, zone: undefined });

test.each([
  [5, "5 s"],
  [90, "90 s"],
  [600, "10 min"],
  [28_800, "8 h"],
  [86_400, "1 day"],
  [172_800, "2 days"],
  [0, "0 s"],
])("durationWords(%d) is %s", (seconds, words) => {
  expect(durationWords(seconds)).toBe(words);
});

test("countWords writes a percentage without a space and one of a unit in the singular", () => {
  const runs = { one: "run", many: "runs" };
  expect([
    countWords(10, { one: "%", many: "%" }),
    countWords(50, runs),
    countWords(1, runs),
  ]).toEqual(["10%", "50 runs", "1 run"]);
});

test.each([
  [400, "under 1 s"],
  [42_000, "42 s"],
  [180_000, "3 min"],
  [184_000, "3 min 4 s"],
  [3_900_000, "1 h 5 min"],
  [7_200_000, "2 h"],
])("lengthWords(%d) is %s", (milliseconds, words) => {
  expect(lengthWords(milliseconds)).toBe(words);
});

test.each([
  ["this afternoon, for 04:00 next morning", at(16, 14), local(17, 4), "Tonight at 04:00"],
  ["this evening, for 22:30", at(16, 19), local(16, 22, 30), "Tonight at 22:30"],
  ["in the small hours, for 04:00", at(17, 2), local(17, 4), "Tonight at 04:00"],
  ["in the morning, for 10:15", at(16, 8), local(16, 10, 15), "Today at 10:15"],
  ["this evening, for tomorrow evening", at(16, 20), local(17, 20), "Tomorrow at 20:00"],
  ["this morning, for 04:00 in three days", at(16, 9), local(19, 4), "Mon at 04:00"],
  ["for a run due already", at(16, 9), local(16, 8), "Due now"],
])("atWords %s", (_case, now, iso, words) => {
  expect(atWords(iso, browser(now))).toBe(words);
});

test.each([
  [local(16, 14, 25), "In 25 min"],
  [local(16, 20), "In 6 h"],
  [local(16, 14, 59), "In 59 min"],
  [local(19, 14), "In 3 days"],
  [local(16, 13), "Due now"],
])("inWords(%s) from 14:00 is %s", (iso, words) => {
  expect(inWords(iso, at(16, 14))).toBe(words);
});

test("sinceWords gives the time today and the date before today", () => {
  expect([
    sinceWords(local(16, 9, 5), browser(at(16, 14))),
    sinceWords(local(15, 21, 31), browser(at(16, 14))),
  ]).toEqual(["09:05", "Oct 15, 21:31"]);
});

const afternoonInNewYork = { now: new Date("2026-10-16T18:00:00Z"), zone: "America/New_York" };

test("time words read the clock in the zone given, whatever the browser's", () => {
  expect([
    atWords("2026-10-17T08:00:00Z", afternoonInNewYork),
    atWords("2026-10-17T08:00:00Z", { now: afternoonInNewYork.now, zone: "Asia/Tokyo" }),
    sinceWords("2026-10-16T13:05:00Z", afternoonInNewYork),
    sinceWords("2026-10-16T01:31:00Z", afternoonInNewYork),
  ]).toEqual(["Tonight at 04:00", "Today at 17:00", "09:05", "Oct 15, 21:31"]);
});

test("a zone Intl doesn't know by name falls back to the browser's", () => {
  expect(
    ["Europe/London", "UTC", "BST", "CEST", "Mars/Olympus"].map((name) => knownZone(name)),
  ).toEqual(["Europe/London", "UTC", undefined, undefined, undefined]);
});
