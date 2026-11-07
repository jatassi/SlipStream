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
    pillsOf(summary({ met: 3, failed: 1, downloading: 2, upgradable: 1, missing: 2 })),
  ).toEqual({
    pills: [
      { tone: "plain", words: "3 of 9 met" },
      { tone: "fail", words: "1 failed" },
      { tone: "live", words: "2 in flight" },
    ],
    rest: ["1 upgrade available", "2 missing"],
  });
});

test("every cell met reads as met", () => {
  expect(pillsOf(summary({ met: 2 }))).toEqual({
    pills: [{ tone: "met", words: "2 of 2 met" }],
    rest: [],
  });
});

test("a Work with no cells shows no pills", () => {
  expect(pillsOf(summary({}))).toEqual({ pills: [], rest: [] });
});
