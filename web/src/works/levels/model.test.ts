import { expect, test } from "vitest";

import type {
  FilesState,
  TargetsWants,
  WorksContentType,
  WorksItem,
  WorksWorkView,
} from "../../api/gen/model";
import { anthologies, longShelf, storiesPerVolume } from "../../test/anthology";
import { blackAndChrome, films } from "../../test/profiles";
import { shelfCells } from "../../test/shelf-cells";
import { keeper, starter } from "../../test/targets";
import { type LevelView, levelContext, levelsOf } from "./model";

const anyCut: TargetsWants = {
  profileId: films.id,
  profileName: films.name,
  editions: { kind: "any" },
  from: "profile",
};

const context = (type: WorksContentType = anthologies, wants = anyCut) =>
  levelContext({ type, profile: films, targets: [starter, keeper], wants, cuts: [blackAndChrome] });

const volumeOf = (item: WorksItem) => longShelf.levels.findIndex(({ id }) => id === item.levelId);

/** Volume 1 has 6 met and the rest missing, Volume 2 is all met, Volume 3 unreleased, the rest met. */
const stateOf = (item: WorksItem): FilesState => {
  const volume = volumeOf(item);
  if (volume === 0) {
    return item.position <= 6 ? "met" : "missing";
  }
  return volume === 2 ? "unreleased" : "met";
};

const named = (levels: readonly LevelView[], label: string) => {
  const level = levels.find((view) => view.name === label);
  if (level === undefined) {
    throw new Error(`no level ${label}`);
  }
  return level;
};

test("each level counts the Items that count and opens while one is wanted and not met", () => {
  const levels = levelsOf(context(), longShelf, shelfCells(longShelf, stateOf));
  const first = named(levels, "Volume 1");
  expect(first.tally).toEqual({ met: 6, of: storiesPerVolume });
  expect(first.counts).toEqual({ met: 6, missing: storiesPerVolume - 6 });
  expect(first.opens).toBe(true);
  expect(first.monitor).toBe(true);
  expect(named(levels, "Volume 2").opens).toBe(false);
  expect(named(levels, "Volume 3").opens).toBe(false);
});

test("levels run in tree order with the aside level last, and nobody monitoring the aside counts nothing there", () => {
  const levels = levelsOf(
    context(),
    longShelf,
    shelfCells(longShelf, () => "missing"),
  );
  expect(levels.map((view) => view.name)).toEqual([
    ...Array.from({ length: 24 }, (_, i) => `Volume ${i + 1}`),
    "Extras",
  ]);
  const extras = named(levels, "Extras");
  expect(extras.tally).toBeNull();
  expect(extras.counts).toEqual({});
  expect(extras.monitor).toBe(false);
  expect(extras.opens).toBe(false);
});

test("a level's switch is mixed while only some of its live Items are monitored", () => {
  const work: WorksWorkView = {
    ...longShelf,
    items: longShelf.items.map((item) =>
      volumeOf(item) === 0 && item.position === 3 ? { ...item, monitored: false } : item,
    ),
  };
  const levels = levelsOf(
    context(),
    work,
    shelfCells(work, () => "missing"),
  );
  expect(named(levels, "Volume 1").monitor).toBe("mixed");
  expect(named(levels, "Volume 1").tally).toEqual({ met: 0, of: storiesPerVolume - 1 });
});

test("a run of unreleased Items folds into one line built from the Item noun and the codes", () => {
  const work: WorksWorkView = {
    ...longShelf,
    items: longShelf.items.map((item) =>
      volumeOf(item) === 0 && item.position >= 4 && item.position <= 7
        ? { ...item, availability: { date: `2026-10-${10 + item.position}`, basis: "" } }
        : item,
    ),
  };
  const cells = shelfCells(work, (item) =>
    volumeOf(item) === 0 && item.position >= 4 && item.position <= 7 ? "unreleased" : "met",
  );
  const entries = named(levelsOf(context(), work, cells), "Volume 1").entries;
  expect(entries).toHaveLength(storiesPerVolume - 3);
  expect(entries[3]).toMatchObject({
    kind: "fold",
    fold: { words: "Stories 4 to 7 come out October 14 to October 17" },
  });
});

const contextOnce = context();

test("an unchanged level keeps its view and a changed one is rebuilt", () => {
  const cells = shelfCells(longShelf, stateOf);
  const before = levelsOf(contextOnce, longShelf, cells);
  const changed = longShelf.items.find((item) => volumeOf(item) === 1);
  const work: WorksWorkView = {
    ...longShelf,
    items: longShelf.items.map((item) => (item === changed ? { ...item, monitored: false } : item)),
  };
  const after = levelsOf(contextOnce, work, cells);
  expect(after[0]).toBe(before[0]);
  expect(after[1]).not.toBe(before[1]);
  expect(after[1]?.monitor).toBe("mixed");
  expect(after[2]).toBe(before[2]);
});

test("a cut nobody in a level has been seen in is a narrow column there, and one others have is a dotted line", () => {
  const wants: TargetsWants = {
    ...anyCut,
    editions: { kind: "cuts", cuts: [blackAndChrome.key] },
  };
  const cells = shelfCells(
    longShelf,
    () => "missing",
    (item) => (volumeOf(item) === 0 && item.position > 1 ? [blackAndChrome.key] : []),
  );
  const levels = levelsOf(context(anthologies, wants), longShelf, cells);
  const first = named(levels, "Volume 1");
  expect(first.columns).toEqual([
    { edition: blackAndChrome.key, name: blackAndChrome.name, seen: true },
  ]);
  const [one, two] = first.entries.map((entry) => (entry.kind === "item" ? entry.item : undefined));
  expect(one?.groups).toEqual([{ kind: "unseen" }]);
  expect(two?.groups).toEqual([{ kind: "cells", cells: [cells.cells[0]] }]);
  expect(named(levels, "Volume 2").columns[0]?.seen).toBe(false);
});

const withdrawnLevel = (index: number) => index === 1 || index === 2;

test("a withdrawn Item with a file sits after its level's live Items, and Items withdrawn with their levels trail in one section that counts them", () => {
  const [gone, plain] = longShelf.items.filter((item) => volumeOf(item) === 0);
  const work: WorksWorkView = {
    ...longShelf,
    levels: longShelf.levels.map((level, index) =>
      withdrawnLevel(index) ? { ...level, withdrawnAt: "2026-10-01T00:00:00Z" } : level,
    ),
    items: longShelf.items.map((item) =>
      item === gone || item === plain || withdrawnLevel(volumeOf(item))
        ? { ...item, withdrawnAt: "2026-10-01T00:00:00Z" }
        : item,
    ),
  };
  const cells = shelfCells(work, (item) => {
    if (item.id === plain?.id || volumeOf(item) === 2) {
      return item.position === 1 ? "met" : "missing";
    }
    return "met";
  });
  const levels = levelsOf(context(), work, cells);
  const first = named(levels, "Volume 1");
  expect(first.gone.map((view) => view.item.id)).toEqual([gone?.id]);
  expect(first.entries).toHaveLength(storiesPerVolume - 2);
  expect(levels.filter(({ withdrawn }) => withdrawn)).toHaveLength(1);
  const trailing = levels.at(-1);
  expect(trailing?.name).toBe("No longer listed by Fixture source");
  expect(trailing?.gone).toHaveLength(storiesPerVolume + 1);
  expect(trailing?.counts).toEqual({ met: storiesPerVolume + 1 });
  expect(levels.map(({ name }) => name)).not.toContain("Volume 2");
});

test("a level shows its level-tier poster", () => {
  const work: WorksWorkView = {
    ...longShelf,
    levels: longShelf.levels.map((level, index) =>
      index === 0
        ? { ...level, artwork: [{ kind: "spine", url: "/api/v1/artwork/spine-1" }] }
        : level,
    ),
  };
  const levels = levelsOf(
    context(),
    work,
    shelfCells(work, () => "met"),
  );
  expect(levels[0]?.poster).toBe("/api/v1/artwork/spine-1");
  expect(levels[1]?.poster).toBeUndefined();
});

const records: WorksContentType = {
  ...anthologies,
  id: "records",
  name: "Records",
  work: { one: "Record", many: "Records" },
  levels: [{ one: "Disc", many: "Discs" }],
  aside: null,
  item: { one: "Track", many: "Tracks" },
  fields: anthologies.fields,
  artwork: [],
};

test("a music-like type's levels and folds read by its own nouns and codes", () => {
  const [disc] = longShelf.levels;
  const album: WorksWorkView = {
    ...longShelf,
    contentType: records.id,
    levels: disc === undefined ? [] : [{ ...disc, label: "Disc 1", aside: false }],
    items: longShelf.items.slice(0, 10).map((item): WorksItem => ({
      id: item.id,
      levelId: item.levelId,
      key: item.key,
      position: item.position,
      withdrawnAt: null,
      monitored: true,
      aside: false,
      ratings: [],
      artwork: [],
      label: `Track ${item.position}`,
      fields: [{ key: "code", kind: "text", text: String(item.position) }],
      availability:
        item.position >= 3 && item.position <= 7
          ? { date: "2027-03-01", basis: "" }
          : item.availability,
    })),
  };
  const cells = shelfCells(album, (item) =>
    item.position >= 3 && item.position <= 7 ? "unreleased" : "met",
  );
  const [level] = levelsOf(context(records), album, cells);
  expect(level?.name).toBe("Disc 1");
  expect(
    level?.entries.map((entry) => (entry.kind === "item" ? entry.item.code : entry.fold.words)),
  ).toEqual(["1", "2", "Tracks 3 to 7 come out March 1", "8", "9", "10"]);
});
