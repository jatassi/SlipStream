import { expect, test } from "vitest";

import type { FilesCell, TargetsWants } from "../../api/gen/model";
import { orchardItemId } from "../../test/library";
import { blackAndChrome, ultra } from "../../test/profiles";
import { keeper, starter } from "../../test/targets";
import { matrixOf } from "./matrix-model";

const cell = (edition: string, targetId: string): FilesCell => ({
  itemId: orchardItemId,
  edition,
  targetId,
  targetName: targetId === keeper.id ? keeper.name : starter.name,
  wanted: true,
  state: "missing",
});

const ownCuts: TargetsWants = {
  profileId: ultra.id,
  profileName: ultra.name,
  editions: { kind: "cuts", cuts: ["theatrical", blackAndChrome.key] },
  from: "work",
};

const cuts = [
  { key: "theatrical", name: "Theatrical", aliases: [], origin: "builtin" as const },
  blackAndChrome,
];

test("the matrix keeps list order for its Editions and Profile order for its Targets", () => {
  const matrix = matrixOf({
    cells: [cell("theatrical", starter.id), cell("theatrical", keeper.id)],
    wants: ownCuts,
    profile: ultra,
    targets: [starter, keeper],
    cuts,
  });
  expect(matrix.targets).toEqual([
    { id: keeper.id, name: "2160p HDR", sourcing: "Downloaded" },
    { id: starter.id, name: "1080p", sourcing: "Downloaded" },
  ]);
  const [theatrical, unseen] = matrix.rows;
  expect(theatrical).toMatchObject({ kind: "edition", edition: "theatrical", name: "Theatrical" });
  expect(
    theatrical?.kind === "edition" && theatrical.cells.map((view) => view.cell?.targetId),
  ).toEqual([keeper.id, starter.id]);
  expect(unseen).toEqual({ kind: "unseen", edition: blackAndChrome.key, name: "Black and Chrome" });
});

test("a Work wanting any cut has one Any cut row", () => {
  const matrix = matrixOf({
    cells: [cell("any", starter.id)],
    wants: { ...ownCuts, editions: { kind: "any" }, from: "profile" },
    profile: ultra,
    targets: [starter, keeper],
    cuts,
  });
  expect(matrix.rows.map((row) => [row.kind, row.name])).toEqual([["edition", "Any cut"]]);
});
