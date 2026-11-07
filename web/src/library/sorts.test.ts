import { expect, test } from "vitest";

import { ctx, scope, view, works } from "../test/films";
import { arrange } from "./arrange";
import type { ArrangeContext, LibraryView, SortKey } from "./view";

const sectionsOf = (change: Partial<LibraryView>, context: ArrangeContext = ctx) => {
  const { sections, order } = arrange(works, scope, view(change), context);
  return sections.map(({ section, start, count }) => [
    section.label,
    order.slice(start, start + count).join(""),
  ]);
};

const expected: Record<SortKey, (readonly [string, string])[]> = {
  title: [
    ["#", "d"],
    ["A", "a"],
    ["D", "e"],
    ["H", "b"],
    ["Z", "c"],
  ],
  added: [
    ["This month", "ea"],
    ["Last month", "b"],
    ["Earlier this year", "d"],
    ["2025", "c"],
  ],
  available: [
    ["Later", "d"],
    ["Next month", "a"],
    ["Later this month", "b"],
    ["This month", "e"],
    ["No date", "c"],
  ],
  size: [
    ["Over 50 GB", "a"],
    ["10 to 50 GB", "b"],
    ["Under 10 GB", "d"],
    ["No files", "ec"],
  ],
  status: [
    ["Failed", "c"],
    ["Upgrade available", "b"],
    ["Missing", "d"],
    ["Met", "a"],
    ["Unreleased", "e"],
  ],
  profile: [
    ["Archive", "ac"],
    ["Films", "deb"],
  ],
  root: [
    ["/media/films", "dac"],
    ["No library root", "eb"],
  ],
  monitored: [
    ["Monitored", "dae"],
    ["Partly monitored", "b"],
    ["Not monitored", "c"],
  ],
  rating: [
    ["8 and up", "a"],
    ["6 to 8", "b"],
    ["4 to 6", "d"],
    ["No rating", "ec"],
  ],
  "field.year": [
    ["2021", "e"],
    ["2019", "d"],
    ["2007", "c"],
    ["2001", "a"],
    ["1995", "b"],
  ],
  "field.runtime": [
    ["2h to 3h", "bce"],
    ["1h to 2h", "a"],
    ["No runtime", "d"],
  ],
  "field.certification": [
    ["PG-13", "c"],
    ["R", "ab"],
    ["No certification", "de"],
  ],
};

test("every sort cuts the Library into sections labelled and ordered by the value it sorts by", () => {
  expect(scope.sorts.map(({ key }) => key)).toEqual(Object.keys(expected));
  for (const { key } of scope.sorts) {
    expect(sectionsOf({ sort: key }), key).toEqual(expected[key]);
  }
});

test("Reverse flips a sort's own direction and still puts the Works lacking its value last", () => {
  expect(sectionsOf({ sort: "available", reverse: true })).toEqual([
    ["This month", "e"],
    ["Later this month", "b"],
    ["Next month", "a"],
    ["Later", "d"],
    ["No date", "c"],
  ]);
});

test("a date to come is never read as past, in the month it falls in or across a new year", () => {
  const january = { ...ctx, today: new Date(2027, 0, 10) };
  expect(sectionsOf({ sort: "available" }, january)).toEqual([
    ["This year", "d"],
    ["2026", "abe"],
    ["No date", "c"],
  ]);
  const early = { ...ctx, today: new Date(2026, 9, 1) };
  expect(sectionsOf({ sort: "available" }, early)).toEqual([
    ["Later", "d"],
    ["Next month", "a"],
    ["Later this month", "be"],
    ["No date", "c"],
  ]);
});
