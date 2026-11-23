import { expect, test } from "vitest";

import { countWords, durationWords, lengthWords, spanWords } from "./format";

test.each([
  [5, "5 s"],
  [600, "10 min"],
  [28_800, "8 h"],
  [86_400, "1 day"],
  [172_800, "2 days"],
  [94_608_000, "1,095 days"],
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
  [30, "30 s"],
  [207, "3 min 27 s"],
  [720, "12 min"],
  [3599.5, "1 h"],
  [7_200, "2 h"],
  [15_120, "4 h 12 min"],
  [15_121, "4 h 13 min"],
  [86_370, "1 day"],
  [273_600, "3 days 4 h"],
  [273_601, "3 days 5 h"],
  [345_600, "4 days"],
])("spanWords(%d) is %s, rounded up to two units", (seconds, words) => {
  expect(spanWords(seconds)).toBe(words);
});
