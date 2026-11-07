import { expect, test } from "vitest";

import type { FilesState } from "../api/gen/model";
import { cinema, libraryRow, orchardRow } from "../test/library";
import { parseRow } from "./rows";
import { type MarksFor, marksOf, statusMarks } from "./status";

test("a state the API adds fails the status record until it has a mark", () => {
  // @ts-expect-error held has no mark, so the record doesn't cover the grown enum
  const grown: MarksFor<FilesState | "held"> = statusMarks;
  expect(Object.keys(grown)).toEqual([
    "unreleased",
    "missing",
    "downloading",
    "failed",
    "upgradable",
    "met",
  ]);
});

const words = (row: Parameters<typeof parseRow>[0]) =>
  marksOf(parseRow(row, cinema)).map(({ word }) => word);

test("a Work's marks are its status, then Not monitored, then gone from its type's source", () => {
  expect(words(orchardRow)).toEqual(["Missing"]);
  expect(
    words(
      libraryRow({
        id: "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b",
        key: "9000002",
        label: "Orchard Harvest",
        status: "met",
        monitored: "none",
        goneUpstreamAt: "2026-10-20T09:00:00Z",
      }),
    ),
  ).toEqual(["Met", "Not monitored", "Gone from TMDB"]);
});

test("a Work with no cell and some Items monitored has no mark", () => {
  expect(words({ ...orchardRow, status: null, monitored: "some" })).toEqual([]);
});
