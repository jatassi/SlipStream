import { http } from "msw";
import { expect, test } from "vitest";
import { userEvent } from "vitest/browser";

import type { GrabsManualSearch } from "../api/gen/model";
import { problem } from "../test/handlers";
import { liveServer } from "../test/live-server";
import { filmPlan, manualSearchId, runId } from "../test/manual";
import { manualServer } from "../test/manual-server";
import { openSheet, routedWords, sheetNamed, twoGrabs, workId } from "../test/manual-sheet";
import { renderApp } from "../test/render-app";
// oxlint-disable-next-line import/max-dependencies -- the sheet reads the plan, the Targets and the live stream
import { starter, targetHandlers } from "../test/targets";
import { openManualSearch } from "./sheet-state";

test("the sheet leads with the headline, then each Grab in words, the cell nothing fills, the folded line and the Searching line", async () => {
  const server = manualServer({ plan: () => filmPlan });
  const live = liveServer();
  renderApp({
    path: "/library",
    handlers: [...live.handlers, ...server.handlers, ...targetHandlers()],
  });
  const cell = {
    itemId: "0199f3a1-5c2e-7d4f-8a6b-1c3e5a7c9eab",
    targetId: starter.id,
    edition: "any",
  };
  openManualSearch({ workId, scope: { cell } });
  const sheet = sheetNamed("The Quiet Orchard (2021)");
  await expect.element(sheet).toBeVisible();
  expect(server.started).toEqual([{ cell }]);

  await expect.element(sheet.getByText(filmPlan.plan.headline)).toBeVisible();
  await expect.element(sheet.getByText("SlipStream searched 2 Indexers.")).toBeVisible();
  await expect.element(sheet.getByText(/^Started .+ ago by Ada Okafor$/u)).toBeVisible();
  const grabs = sheet.getByRole("list", { name: "Grabs" });
  await expect.element(grabs.getByText("WEBDL-1080p · DD+ 5.1 · FLUX · 4.0 GB")).toBeVisible();
  await expect
    .element(grabs.getByText("It downloads 1080p for The Quiet Orchard (2021)."))
    .toBeVisible();
  await expect.element(grabs.getByText(routedWords)).toBeVisible();
  await expect
    .element(grabs.getByRole("checkbox", { name: "WEBDL-1080p · DD+ 5.1 · FLUX · 4.0 GB" }))
    .toBeChecked();
  const gaps = sheet.getByRole("list", { name: "Files with no release found" });
  await expect
    .element(gaps.getByText("Missing: 2160p HDR for The Quiet Orchard (2021)."))
    .toBeVisible();
  await expect.element(gaps.getByText("No Indexer listed a 2160p release.")).toBeVisible();
  await expect.element(sheet.getByText("SlipStream wouldn't grab 4 more releases.")).toBeVisible();
  await expect.element(sheet.getByRole("button", { name: "Grab 1" })).toBeEnabled();
});

test("red marks only an Indexer that failed and a group of Releases with a fault", async () => {
  const faulty: GrabsManualSearch = {
    ...filmPlan,
    indexers: filmPlan.indexers.map((line, at) =>
      at === 0 ? line : { ...line, sentence: "TorrentLeech isn't responding.", failed: true },
    ),
    folded: {
      line: "SlipStream wouldn't grab 4 more releases.",
      groups: [
        { words: "3 are 720p, which no Target here accepts.", fault: false },
        { words: "1 has a .torrent that couldn't be read.", fault: true },
      ],
    },
  };
  openSheet(manualServer({ plan: () => faulty }));
  const sheet = sheetNamed("The Quiet Orchard (2021)");
  await userEvent.click(sheet.getByRole("button", { name: "Details" }).first());
  const indexers = sheet.getByRole("list", { name: "Indexers" });
  await expect.element(indexers.getByRole("listitem").nth(1).getByText("Failed")).toBeVisible();
  await expect
    .element(indexers.getByRole("listitem").first().getByText("Failed"))
    .not.toBeInTheDocument();

  await userEvent.click(sheet.getByRole("button", { name: "Details" }).nth(2));
  const folded = sheet.getByRole("list", { name: "Releases SlipStream wouldn't grab" });
  await expect
    .element(folded.getByRole("listitem").nth(1))
    .toMatchTextContent(/1 has a \.torrent/u);
  await expect.element(folded.getByRole("listitem").first()).toMatchTextContent(/3 are 720p/u);
});

test("a search another Admin replaced says so in place of the Grab button", async () => {
  const replaced = "A newer Manual search replaced this list. Open Manual search again to see it.";
  openSheet(manualServer({ plan: () => filmPlan }), {
    handlers: [
      http.post("*/api/v1/works/:workId/manual-search/grabs", () =>
        problem(409, "manual_search_replaced", "This Manual search was replaced", replaced),
      ),
    ],
  });
  const sheet = sheetNamed("The Quiet Orchard (2021)");
  await userEvent.click(sheet.getByRole("button", { name: "Grab 1" }));
  await expect.element(sheet.getByRole("alert")).toHaveTextContent(replaced);
  await expect.element(sheet.getByRole("button", { name: /^Grab/u })).not.toBeInTheDocument();
  await expect
    .element(sheet.getByRole("button", { name: "Open Manual search again" }))
    .toBeVisible();
});

test("a Manual search's event reads the plan again", async () => {
  let rows = 1;
  const server = manualServer({
    plan: (pins) => {
      const plan = twoGrabs(pins);
      return { ...plan, plan: { ...plan.plan, rows: plan.plan.rows.slice(0, rows) } };
    },
  });
  const live = openSheet(server);
  const sheet = sheetNamed("Season 1 of The Long Shelf");
  await expect.element(sheet.getByRole("button", { name: "Grab 1" })).toBeVisible();
  rows = 2;
  live.clients[0]?.send({
    event: "grabs.search-changed",
    data: { workId, manualSearchId, searchId: runId },
  });
  await expect
    .element(sheet.getByRole("checkbox", { name: "WEBDL-1080p · FLUX · 4.0 GB" }))
    .toBeChecked();
  await expect.element(sheet.getByRole("button", { name: "Grab 2" })).toBeVisible();
});
