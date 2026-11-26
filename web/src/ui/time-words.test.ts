import { expect, test } from "vitest";

import { type Clock, knownZone } from "./clock";
import { type TimeForm, isToday, momentWords, placeOf, timeForms, whenWords } from "./time-words";

const local = (day: number, hours: number, minutes = 0) =>
  new Date(2026, 9, day, hours, minutes).toISOString();

const own = (day: number, hours: number, minutes = 0): Clock => ({
  now: new Date(2026, 9, day, hours, minutes),
  zone: undefined,
});

const words = (form: TimeForm, at: string, clock: Clock = own(16, 14)) =>
  timeForms[form](placeOf(at, clock));

test.each([
  ["today", local(16, 9, 5), "9:05 AM"],
  ["yesterday", local(15, 21, 31), "Oct 15, 9:31 PM"],
  ["tomorrow", local(17, 8), "Oct 17, 8:00 AM"],
  ["another year", new Date(2025, 9, 15, 21, 31).toISOString(), "Oct 15, 2025, 9:31 PM"],
])("a clock time shows its day only when it isn't today: %s", (_case, at, shown) => {
  expect(words("clock", at)).toBe(shown);
});

test("a time and a stamp leave the day to the heading or the log", () => {
  expect([words("time", local(15, 21, 31)), words("stamp", local(16, 9, 5))]).toEqual([
    "9:31 PM",
    "09:05:00.000",
  ]);
});

test.each([
  ["this afternoon, for 04:00 next morning", own(16, 14), local(17, 4), "Tonight at 4:00 AM"],
  ["this evening, for 22:30", own(16, 19), local(16, 22, 30), "Tonight at 10:30 PM"],
  ["in the small hours, for 04:00", own(17, 2), local(17, 4), "Tonight at 4:00 AM"],
  ["in the morning, for 10:15", own(16, 8), local(16, 10, 15), "Today at 10:15 AM"],
  ["this evening, for tomorrow evening", own(16, 20), local(17, 20), "Tomorrow at 8:00 PM"],
  ["this morning, for 04:00 in three days", own(16, 9), local(19, 4), "Mon at 4:00 AM"],
  ["this morning, for a run in two weeks", own(16, 9), local(30, 4), "Oct 30 at 4:00 AM"],
  ["for a run due already", own(16, 9), local(16, 8), "Due now"],
])("a coming time %s", (_case, clock, at, shown) => {
  expect(words("due", at, clock)).toBe(shown);
});

test.each([
  [local(16, 14, 25), "In 25 min"],
  [local(16, 20), "In 6 h"],
  [local(19, 14), "In 3 days"],
  [local(16, 13), "Due now"],
])("a countdown to %s from 14:00 is %s", (at, shown) => {
  expect(words("countdown", at)).toBe(shown);
});

test.each([
  [new Date(2026, 9, 16, 13, 59, 30).toISOString(), "just now"],
  [local(16, 13, 48), "12 minutes ago"],
  [local(16, 11), "3 hours ago"],
  [local(14, 13), "2 days ago"],
])("an age of %s at 14:00 is %s", (at, shown) => {
  expect(words("age", at)).toBe(shown);
});

test.each([
  ["at", local(16, 9, 12), "at 9:12 AM"],
  ["at", local(15, 9, 12), "yesterday at 9:12 AM"],
  ["at", local(17, 2), "tonight at 2:00 AM"],
  ["at", local(17, 9), "tomorrow at 9:00 AM"],
  ["at", local(12, 9, 12), "on Mon at 9:12 AM"],
  ["at", local(3, 9, 12), "on Oct 3 at 9:12 AM"],
  ["next", local(16, 18), "at 6:00 PM"],
  ["next", local(19, 14), "on Mon at 2:00 PM"],
  ["next", local(16, 13), "now"],
  ["in", local(16, 14, 5), "in 5 min"],
  ["in", local(16, 13), "now"],
  ["since", local(16, 9, 5), "since 9:05 AM"],
  ["since", local(15, 9, 5), "since yesterday at 9:05 AM"],
  ["since", local(3, 9, 5), "since Oct 3 at 9:05 AM"],
  ["sinceDay", local(16, 9, 5), "today"],
  ["sinceDay", local(15, 9, 5), "since yesterday"],
  ["sinceDay", local(12, 9, 5), "since Mon"],
  ["sinceDay", local(3, 9, 5), "since Oct 3"],
  ["until", local(16, 18), "until 6:00 PM"],
  ["until", local(17, 2), "until tonight at 2:00 AM"],
  ["until", local(19, 14), "until Mon at 2:00 PM"],
  ["on", local(16, 9), "today"],
  ["on", local(15, 9), "yesterday"],
  ["on", local(3, 9), "on Oct 3"],
  ["heading", local(16, 9), "Today"],
  ["heading", local(15, 9), "Yesterday"],
  ["heading", local(13, 9), "Tuesday 13 October"],
  ["heading", new Date(2025, 9, 14, 9).toISOString(), "Tuesday 14 October 2025"],
] as const)("a time in a sentence %s %s reads %s", (form, at, shown) => {
  expect(whenWords(at, form, own(16, 14))).toBe(shown);
});

const afternoonInNewYork = { now: new Date("2026-10-16T18:00:00Z"), zone: "America/New_York" };

test("a clock in a zone reads its days and times there, whatever the person's", () => {
  expect([
    words("due", "2026-10-17T08:00:00Z", afternoonInNewYork),
    words("due", "2026-10-17T08:00:00Z", { now: afternoonInNewYork.now, zone: "Asia/Tokyo" }),
    words("clock", "2026-10-16T13:05:00Z", afternoonInNewYork),
    words("clock", "2026-10-16T01:31:00Z", afternoonInNewYork),
    isToday("2026-10-16T01:31:00Z", afternoonInNewYork),
  ]).toEqual(["Tonight at 4:00 AM", "Today at 5:00 PM", "9:05 AM", "Oct 15, 9:31 PM", false]);
});

test("the whole moment adds the person's own clock when the zone is another's", () => {
  const at = "2026-10-16T13:05:00Z";
  const there = momentWords(at, { now: afternoonInNewYork.now, zone: "Pacific/Chatham" });
  expect([there.length, there[0], momentWords(at, own(16, 14))]).toEqual([
    2,
    "Saturday, October 17, 2026 at 2:50:00 AM GMT+13:45",
    [there[1]],
  ]);
});

test("a zone Intl doesn't know by name is the person's own", () => {
  expect(
    ["Europe/London", "UTC", "BST", "CEST", "Mars/Olympus"].map((name) => knownZone(name)),
  ).toEqual(["Europe/London", "UTC", undefined, undefined, undefined]);
});
