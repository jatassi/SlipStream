import { expect, test } from "vitest";

import { anthologies } from "../../test/anthology";
import { type Foldable, foldWords, foldsOf } from "./fold";

const stories = anthologies;
const tracks = { item: { one: "Track", many: "Tracks" }, releases: "" };
const episodes = { item: { one: "Episode", many: "Episodes" }, releases: "air" };

const story = (n: number, date: string | null, unreleased = true): Foldable => ({
  code: `V01S${String(n).padStart(2, "0")}`,
  date,
  unreleased,
});

test("a run of unreleased Items reads by the Item noun and the number its codes differ in", () => {
  expect(
    foldWords(stories, [
      story(4, "2026-10-12"),
      story(5, "2026-10-19"),
      story(6, "2026-10-26"),
      story(7, "2026-11-02"),
    ]),
  ).toBe("Stories 4 to 7 come out October 12 to November 2");
});

test("a type's own verb for coming out replaces come out", () => {
  expect(
    foldWords(episodes, [
      { code: "S01E04", date: "2026-10-12", unreleased: true },
      { code: "S01E07", date: "2026-11-02", unreleased: true },
    ]),
  ).toBe("Episodes 4 and 7 air October 12 to November 2");
});

test("two Items read with and, and one shared date reads once", () => {
  expect(foldWords(stories, [story(4, "2026-10-12"), story(5, "2026-10-12")])).toBe(
    "Stories 4 and 5 come out October 12",
  );
});

test("a run with no dates says so", () => {
  expect(foldWords(stories, [story(8, null), story(9, null), story(10, null)])).toBe(
    "Stories 8 to 10 have no availability date yet",
  );
});

test("codes that differ before their last number read in full", () => {
  expect(
    foldWords(stories, [
      { code: "V01S49", date: "2026-12-28", unreleased: true },
      { code: "V02S01", date: "2027-01-04", unreleased: true },
    ]),
  ).toBe("Stories V01S49 and V02S01 come out December 28, 2026 to January 4, 2027");
});

const track = (n: number): Foldable => ({ code: String(n), date: "2027-03-01", unreleased: true });

test("a music-like type's bare track numbers fold the same way", () => {
  expect(foldWords(tracks, [track(3), track(4), track(5), track(6), track(7)])).toBe(
    "Tracks 3 to 7 come out March 1",
  );
});

test("released Items stay rows, a lone unreleased Item stays a row and dated runs fold apart from undated ones", () => {
  const items = [
    story(1, "2026-09-01", false),
    story(2, "2026-10-12"),
    story(3, "2026-09-20", false),
    story(4, "2026-10-19"),
    story(5, "2026-10-26"),
    story(6, null),
    story(7, null),
  ];
  expect(
    foldsOf(stories, items).map((entry) =>
      entry.kind === "item" ? entry.item.code : entry.fold.words,
    ),
  ).toEqual([
    "V01S01",
    "V01S02",
    "V01S03",
    "Stories 4 and 5 come out October 19 to October 26",
    "Stories 6 and 7 have no availability date yet",
  ]);
});
