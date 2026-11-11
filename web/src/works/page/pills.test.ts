import { expect, test } from "vitest";

import { pillsOf } from "./pills";

const summary = (cells: Parameters<typeof pillsOf>[0]["cells"]) => ({
  status: "missing" as const,
  cells,
  items: 1,
  itemStatus: {},
});

test("the met pill counts every cell, failures and downloads are pills, the rest is text", () => {
  expect(
    pillsOf(summary({ met: 3, failed: 1, downloading: 2, upgradable: 1, missing: 2 }), "one"),
  ).toEqual({
    pills: [
      { tone: "plain", words: "3 of 9 met" },
      { tone: "fail", words: "1 failed" },
      { tone: "live", words: "2 downloading" },
    ],
    rest: ["2 missing", "1 upgrade available"],
  });
});

test("every cell met reads as met", () => {
  expect(pillsOf(summary({ met: 2 }), "one")).toEqual({
    pills: [{ tone: "met", words: "2 of 2 met" }],
    rest: [],
  });
});

test("a Work with no cells shows no pills", () => {
  expect(pillsOf(summary({}), "one")).toEqual({ pills: [], rest: [] });
});

test("many Items count the Items that count, as the poster does, not their cells", () => {
  expect(
    pillsOf(
      {
        status: "missing",
        cells: { met: 30, missing: 10 },
        items: 40,
        itemStatus: { met: 12, failed: 1, missing: 7 },
      },
      "many",
    ),
  ).toEqual({
    pills: [
      { tone: "plain", words: "12 of 40 met" },
      { tone: "fail", words: "1 failed" },
    ],
    rest: ["7 missing"],
  });
});
