import { expect, test } from "vitest";

import type { FilesWorkLine } from "../api/gen/model";
import { row } from "../test/renames";
import {
  confirmBody,
  everyTick,
  onlyTicked,
  rowStates,
  rowsNeeded,
  tickFile,
  tickWork,
  tickedCounts,
} from "./plan-model";

const line = (workId: string, rename: number, blocked = 0): FilesWorkLine => ({
  workId,
  contentType: "anthology",
  title: workId,
  digest: `${workId}-digest`,
  rename,
  blocked,
  moving: 0,
});

const swapped = [
  row("e1", "Shelf/Shelf - 1x01.mkv", "Shelf/Shelf - 1x02.mkv", { group: 1 }),
  row("e2", "Shelf/Shelf - 1x02.mkv", "Shelf/Shelf - 1x01.mkv", { group: 1 }),
  row("e3", "Shelf/Shelf - 1x03 Old.mkv", "Shelf/Shelf - 1x03 New.mkv"),
];

test("leaving a file that trades names with another leaves both, and ticking one ticks both", () => {
  const shelf = line("shelf", 3);
  const rows = new Map([["shelf", swapped]]);
  const left = tickFile(everyTick, "shelf", swapped, "e2", false);
  expect(tickedCounts([shelf], left, rows)).toEqual({ files: 1, works: 1 });
  expect(confirmBody([shelf], left, rows)).toEqual([
    { work: "shelf", digest: "shelf-digest", leave: ["e1", "e2"] },
  ]);

  const back = tickFile(left, "shelf", swapped, "e1", true);
  expect(tickedCounts([shelf], back, rows)).toEqual({ files: 3, works: 1 });
  expect(confirmBody([shelf], back, rows)).toEqual([{ work: "shelf", digest: "shelf-digest" }]);
});

test("a file that trades names with a blocked file stays, and isn't counted", () => {
  const rows = [
    row("e1", "a.mkv", "b.mkv", { group: 2 }),
    row("e2", "b.mkv", "a.mkv", {
      group: 2,
      blocked: { reason: "too_long", detail: "The name is too long." },
    }),
    row("e3", "c.mkv", "d.mkv"),
  ];
  expect(Object.fromEntries(rowStates(rows))).toEqual({
    e1: "held",
    e2: "blocked",
    e3: "renames",
  });
  expect(tickedCounts([line("w", 2, 1)], everyTick, new Map([["w", rows]]))).toEqual({
    files: 1,
    works: 1,
  });
});

test("a Work left out whole is confirmed with every file left, once its rows are read", () => {
  const lines = [line("kept", 2), line("left", 1), line("stuck", 0, 1)];
  const ticks = tickWork(everyTick, "left", false);
  expect(tickedCounts(lines, ticks, new Map())).toEqual({ files: 2, works: 1 });
  expect(rowsNeeded(lines, ticks, new Map())).toEqual(["left"]);

  const rows = new Map([["left", [row("f9", "x.mkv", "y.mkv")]]]);
  expect(confirmBody(lines, ticks, rows)).toEqual([
    { work: "kept", digest: "kept-digest" },
    { work: "left", digest: "left-digest", leave: ["f9"] },
  ]);
});

test("ticking a Work back on renames all its files again", () => {
  const lines = [line("shelf", 3)];
  const rows = new Map([["shelf", swapped]]);
  const some = tickFile(everyTick, "shelf", swapped, "e3", false);
  expect(tickedCounts(lines, some, rows).files).toBe(2);
  expect(tickedCounts(lines, tickWork(some, "shelf", true), rows).files).toBe(3);
});

test("after a confirm, only the Works that changed are ticked", () => {
  const lines = [line("a", 1), line("b", 2), line("c", 1)];
  const ticks = onlyTicked(lines, new Set(["b"]));
  expect(tickedCounts(lines, ticks, new Map())).toEqual({ files: 2, works: 1 });
});
