import { expect, test } from "vitest";

import type { WorksContentType } from "../api/gen/model";
import { cinema } from "../test/library";
import { ctx, films, listed, scope, view, works } from "../test/films";
import { arrange, scopeOf } from "./arrange";
import type { LibraryView } from "./view";

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
  expect(all.groups.map(({ key }) => key)).toEqual([
    "status",
    "monitored",
    "profile",
    "root",
    "needs",
  ]);
});
