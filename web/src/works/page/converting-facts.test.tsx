import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type {
  ConversionsConversionRow,
  FilesCellList,
  FilesConvertingCell,
} from "../../api/gen/model";
import { getListConversionsMockHandler } from "../../api/gen/query/conversions/conversions.msw";
import { orchard, orchardId } from "../../test/library";
import { home } from "../../test/nodes";
import { renderApp } from "../../test/render-app";
import { missing } from "../../test/work-cells";
import { workServer } from "../../test/work-server";

const conversionId = "c0de0000-1111-4222-8333-444455556666";

const [hdCell] = missing.cells;
if (hdCell?.state !== "missing") {
  throw new Error("missing leads with a missing 1080p cell");
}
const { state: _missing, ...unconverted } = hdCell;

function converting(stage: FilesConvertingCell["stage"]): FilesCellList {
  const cell: FilesConvertingCell = {
    ...unconverted,
    state: "converting",
    conversionId,
    stage,
    node: { id: home.id, name: home.name },
    file: null,
    wanted: false,
  };
  return { ...missing, cells: [cell] };
}

const row = (
  stage: ConversionsConversionRow["stage"],
  change: Partial<ConversionsConversionRow>,
): ConversionsConversionRow => ({
  kind: "conversion",
  id: conversionId,
  stage,
  lane: "new",
  work: { id: orchardId, title: orchard.label, type: orchard.contentType },
  items: [],
  target: { id: hdCell.targetId, name: hdCell.targetName },
  source: { fileId: null, name: "The.Quiet.Orchard.2021.2160p.mkv", targetId: null, target: "2160p HDR", edition: "any" },
  node: { id: home.id, name: home.name },
  progress: null,
  pausedBy: null,
  wait: null,
  start: null,
  finish: null,
  failure: null,
  attempts: [],
  ...change,
});

const cutRow = () =>
  page.getByRole("table", { name: "Editions and Targets" }).getByRole("row", { name: /^Any cut/u });

test.each([
  {
    name: "a running Conversion shows how far it has run",
    stage: "running",
    listed: row("running", { progress: { percent: 42.6, speed: 3.1, secondsLeft: 600 } }),
    band: "converting",
    shows: async () => {
      await expect
        .element(cutRow().getByRole("progressbar", { name: "Converting 1080p" }))
        .toHaveAttribute("aria-valuenow", "42");
    },
  },
  {
    name: "a queued Conversion names the room its Node lacks",
    stage: "queued",
    listed: row("queued", {
      wait: {
        code: "room",
        node: { id: home.id, name: home.name },
        free: 41 * 1024 ** 3,
        need: 52 * 1024 ** 3,
      },
    }),
    band: "next",
    shows: async () => {
      await expect
        .element(cutRow())
        .toMatchTextContent(new RegExp(`room on ${home.name}.*41`, "u"));
      await expect.element(cutRow().getByRole("progressbar")).not.toBeInTheDocument();
    },
  },
] as const)("$name, from the Work's Conversions", async ({ stage, listed, band, shows }) => {
  const asked: (string | null)[] = [];
  const server = workServer({ startCells: converting(stage) });
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [
      getListConversionsMockHandler(({ request }) => {
        asked.push(new URL(request.url).searchParams.get("work"));
        return {
          counts: { converting: 0, paused: 0, waiting: 0, failed: 0, clearsAt: null },
          attention: [],
          converting: band === "converting" ? [listed] : [],
          next: band === "next" ? [{ lane: "new", rows: [listed] }] : [],
          cursor: null,
        };
      }),
      ...server.handlers,
    ],
  });
  await expect.element(cutRow()).toMatchTextContent(new RegExp(`Converting.*${home.name}`, "u"));
  await shows();
  expect(asked).toContain(orchardId);
});
