import { expect, test } from "vitest";

import type { TargetsWants, WorksItem, WorksWorkView } from "../../api/gen/model";
import { anthologies, longShelf } from "../../test/anthology";
import { films } from "../../test/profiles";
import { shelfCells } from "../../test/shelf-cells";
import { starter } from "../../test/targets";
import { levelContext, levelsOf } from "./model";
import { levelRows } from "./rows";

const wants: TargetsWants = {
  profileId: films.id,
  profileName: films.name,
  editions: { kind: "any" },
  from: "profile",
};
const context = levelContext({
  type: anthologies,
  profile: films,
  targets: [starter],
  wants,
  cuts: [],
});
const volumeOf = (item: WorksItem) => longShelf.levels.findIndex(({ id }) => id === item.levelId);

const [firstStory] = longShelf.items;
const work: WorksWorkView = {
  ...longShelf,
  items: [
    ...(firstStory === undefined ? [] : [{ ...firstStory, withdrawnAt: "2026-10-01T00:00:00Z" }]),
    ...longShelf.items.slice(1),
  ],
};
const levels = levelsOf(
  context,
  work,
  shelfCells(work, (item) => {
    if (volumeOf(item) !== 0 || item.id === firstStory?.id) {
      return "met";
    }
    return item.position >= 40 ? "unreleased" : "missing";
  }),
);

test("a closed level is its header alone", () => {
  const rows = levelRows(levels, () => false);
  expect(rows.map((row) => row.kind)).toEqual(levels.map(() => "header"));
});

test("an open level is its header, Items, folds, withdrawn Items, then its footer", () => {
  const rows = levelRows(levels, (level) => level === levels[0]);
  const [first] = levels;
  const kinds = rows.filter((row) => row.level === first).map((row) => row.kind);
  expect(kinds).toEqual([
    "header",
    ...Array.from({ length: 38 }, () => "item"),
    "fold",
    "withdrawn",
    "footer",
  ]);
  expect(rows.slice(kinds.length).every((row) => row.kind === "header")).toBe(true);
});

test("rows keep their keys when another level opens, and no two share one", () => {
  const one = levelRows(levels, (level) => level === levels[0]);
  const both = levelRows(levels, (level) => level === levels[0] || level === levels[1]);
  expect(both.slice(0, one.length - 23).map((row) => row.key)).toEqual(
    one.slice(0, one.length - 23).map((row) => row.key),
  );
  expect(new Set(both.map((row) => row.key)).size).toBe(both.length);
});
