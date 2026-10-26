import { expect, test } from "vitest";

import type { WorksContentType, WorksFieldSpec, WorksLibraryRow } from "../api/gen/model";
import { cinema, libraryRow } from "../test/library";
import { arrange, scopeOf } from "./arrange";
import { gridRows } from "./grid-layout";
import { gridMove, tableMove } from "./keyboard";
import { type LibraryWork, parseRow } from "./rows";
import { decodeView, encodeView } from "./view-codec";
import { type ArrangeContext, type LibraryView, type SortKey, defaultView } from "./view";

const listed = (
  key: string,
  label: string,
  kind: WorksFieldSpec["kind"],
  list: Partial<WorksFieldSpec["list"]>,
): WorksFieldSpec => ({
  key,
  label,
  tier: "work",
  kind,
  in: ["facts"],
  list: { column: false, sort: false, filter: false, ...list },
  with: null,
  art: null,
});

const films: WorksContentType = {
  ...cinema,
  ratings: ["imdb"],
  fields: [
    ...cinema.fields,
    listed("certification", "Certification", "text", { column: true, sort: true, filter: true }),
    listed("genres", "Genres", "texts", { filter: true }),
  ],
};

const gib = 1024 ** 3;
const minute = 60_000;

const rows: WorksLibraryRow[] = [
  libraryRow({
    id: "a",
    key: "1",
    label: "Amélie",
    addedAt: "2026-10-20T09:00:00Z",
    availableOn: "2026-11-14",
    bytes: 60 * gib,
    files: 1,
    status: "met",
    profileId: "archive",
    rootId: "films",
    monitored: "all",
    rating: { source: "imdb", value: 8.3, scale: 10 },
    values: { year: 2001, runtime: 98 * minute, certification: "R", genres: ["Comedy", "Romance"] },
  }),
  libraryRow({
    id: "b",
    key: "2",
    label: "Heat",
    addedAt: "2026-09-03T09:00:00Z",
    availableOn: "2026-10-30",
    bytes: 12 * gib,
    files: 2,
    status: "upgradable",
    profileId: "films",
    rootId: null,
    monitored: "some",
    rating: { source: "imdb", value: 70, scale: 100 },
    values: { year: 1995, runtime: 170 * minute, certification: "R", genres: ["Crime", "Drama"] },
  }),
  libraryRow({
    id: "c",
    key: "3",
    label: "Zodiac",
    addedAt: "2025-03-01T09:00:00Z",
    availableOn: null,
    bytes: 0,
    files: 0,
    status: "failed",
    profileId: "archive",
    rootId: "films",
    monitored: "none",
    rating: null,
    values: { year: 2007, runtime: 157 * minute, certification: "PG-13", genres: ["Crime"] },
  }),
  libraryRow({
    id: "d",
    key: "4",
    label: "1917",
    addedAt: "2026-02-01T09:00:00Z",
    availableOn: "2027",
    bytes: 2 * gib,
    files: 1,
    status: "missing",
    profileId: "films",
    rootId: "films",
    monitored: "all",
    rating: { source: "imdb", value: 5.5, scale: 10 },
    values: { year: 2019 },
  }),
  libraryRow({
    id: "e",
    key: "5",
    label: "Dune",
    addedAt: "2026-10-26T08:00:00Z",
    availableOn: "2026-10-02",
    bytes: 0,
    files: 0,
    status: "unreleased",
    profileId: "films",
    rootId: null,
    monitored: "all",
    rating: null,
    values: { year: 2021, runtime: 155 * minute },
  }),
];

const works: LibraryWork[] = rows.map((row) => parseRow(row, films));
const types = [films];
const scope = scopeOf(types);

const ctx: ArrangeContext = {
  today: new Date(2026, 9, 26, 12),
  profiles: new Map([
    ["archive", "Archive"],
    ["films", "Films"],
  ]),
  roots: new Map([["films", "/media/films"]]),
  matched: undefined,
};

const view = (change: Partial<LibraryView>): LibraryView => ({ ...defaultView(types), ...change });

const sectionsOf = (change: Partial<LibraryView>, context: ArrangeContext = ctx) => {
  const { sections, order } = arrange(works, scope, view(change), context);
  return sections.map(({ section, start, count }) => [
    section.label,
    order.slice(start, start + count).join(""),
  ]);
};

test("every sort cuts the Library into sections labelled and ordered by the value it sorts by", () => {
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
  expect(scope.sorts.map(({ key }) => key)).toEqual(Object.keys(expected));
  for (const [sort, sections] of Object.entries(expected)) {
    expect(sectionsOf({ sort }), sort).toEqual(sections);
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

const shown = (filters: LibraryView["filters"], matched?: readonly string[]) =>
  arrange(works, scope, view({ filters }), {
    ...ctx,
    matched: matched === undefined ? undefined : new Set(matched),
  }).order.join("");

test("filters keep Works that pass every group, and any of the values picked inside one", () => {
  expect(shown({ status: { kind: "values", values: ["missing", "failed"] } })).toBe("dc");
  expect(
    shown({
      monitored: { kind: "values", values: ["all"] },
      status: { kind: "values", values: ["missing", "failed"] },
    }),
  ).toBe("d");
  expect(shown({ "field.genres": { kind: "values", values: ["Crime", "Comedy"] } })).toBe("abc");
  expect(shown({ "field.year": { kind: "range", min: 2000, max: 2010 } })).toBe("ac");
  expect(shown({ "field.year": { kind: "range", min: 2019, max: undefined } })).toBe("de");
  expect(shown({ root: { kind: "values", values: ["none"] } })).toBe("eb");
  expect(shown({}, ["c", "b"])).toBe("bc");
  expect(shown({ status: { kind: "values", values: ["failed"] } }, ["b"])).toBe("");
});

test("a view goes into the address and comes back the same, and the default view adds nothing", () => {
  const changed: LibraryView = {
    layout: "table",
    size: "large",
    columns: ["status", "size", "field.year"],
    sort: "field.year",
    reverse: true,
    filters: {
      status: { kind: "values", values: ["missing", "failed"] },
      root: { kind: "values", values: ["none"] },
      needs: { kind: "flag" },
      "field.year": { kind: "range", min: 1990, max: undefined },
      "field.genres": { kind: "values", values: ["Crime, Noir", "Drama"] },
    },
    q: "amelie",
  };
  const search = encodeView(changed, types);
  expect(search).toEqual({
    view: "table",
    size: "large",
    cols: "status,size,field.year",
    sort: "field.year",
    rev: 1,
    status: "missing,failed",
    root: "none",
    needs: 1,
    "f.year": "1990..",
    "f.genres": "Crime, Noir~Drama",
    q: "amelie",
  });
  expect(decodeView(search, types)).toEqual(changed);
  expect(encodeView(defaultView(types), types)).toEqual({});
});

test("an address naming what its types no longer declare opens on the rest", () => {
  const search = {
    view: "mosaic",
    sort: "field.genres",
    cols: "field.studio,size",
    "f.studio": "Fennel Films",
    "f.year": "2001",
  };
  expect(decodeView(search, types)).toEqual(
    view({
      columns: ["size"],
      filters: { "field.year": { kind: "range", min: 2001, max: 2001 } },
    }),
  );
});

test("on All, only the fields every type declares alike are offered", () => {
  const zines: WorksContentType = {
    ...cinema,
    id: "zines",
    fields: [listed("year", "Year", "text", { sort: true, filter: true, column: true })],
  };
  const all = scopeOf([films, zines]);
  expect(all.sorts.map(({ key }) => key)).toEqual([
    "title",
    "added",
    "available",
    "size",
    "status",
    "profile",
    "root",
    "monitored",
  ]);
  expect(all.groups.map(({ key }) => key)).toEqual(["status", "monitored", "profile", "root", "needs"]);
});

test("arrow keys move one Work across and keep the column up and down, past section headings", () => {
  const order = ["a", "b", "c", "d", "e", "f", "g"];
  const layout = gridRows(
    {
      order,
      sections: [
        { section: { key: "A", label: "A" }, start: 0, count: 5 },
        { section: { key: "B", label: "B" }, start: 5, count: 2 },
      ],
    },
    3,
  );
  expect(layout.rows.map((row) => (row.kind === "section" ? row.section.label : row.ids.join("")))).toEqual(
    ["A", "abc", "de", "B", "fg"],
  );
  const count = order.length;
  expect(gridMove(layout, 2, "down", count)).toBe(4);
  expect(gridMove(layout, 4, "down", count)).toBe(6);
  expect(gridMove(layout, 3, "down", count)).toBe(5);
  expect(gridMove(layout, 5, "up", count)).toBe(3);
  expect(gridMove(layout, 1, "up", count)).toBe(1);
  expect(gridMove(layout, 2, "right", count)).toBe(3);
  expect(gridMove(layout, 0, "left", count)).toBe(0);
  expect(gridMove(layout, 3, "last", count)).toBe(6);
  expect(tableMove(0, "down", count)).toBe(1);
  expect(tableMove(6, "down", count)).toBe(6);
  expect(tableMove(3, "first", count)).toBe(0);
  expect(tableMove(3, "left", count)).toBe(3);
});
