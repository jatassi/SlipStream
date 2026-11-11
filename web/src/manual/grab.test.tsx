import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { GrabsPlanRow } from "../api/gen/model";
import { downloadClients, homeCard, seedboxCard } from "../test/download-clients";
import {
  blocked,
  blockedChoice,
  filmPlan,
  flux,
  heldGrabId,
  manualSearchId,
  pinnedFilmPlan,
  runId,
} from "../test/manual";
import { manualServer } from "../test/manual-server";
import { openSheet, sheetNamed, twoGrabs, workId } from "../test/manual-sheet";

test("Send all to sets every row without its own client, and a row's Send to keeps its own", async () => {
  const server = manualServer({ plan: twoGrabs });
  const clients = downloadClients({ cards: [homeCard, seedboxCard] });
  openSheet(server, { handlers: clients.handlers });
  const sheet = sheetNamed("Season 1 of The Long Shelf");
  const first = sheet.getByRole("list", { name: "Grabs" }).getByRole("listitem").first();
  await userEvent.click(first.getByRole("button", { name: "Details" }));
  await userEvent.click(first.getByRole("combobox", { name: "Send to" }));
  await userEvent.click(page.getByRole("option", { name: "Built-in client on seedbox" }));
  await expect
    .element(first.getByText("SlipStream sends it to the built-in client on seedbox."))
    .toBeVisible();

  await userEvent.click(sheet.getByRole("combobox", { name: "Send all to" }));
  await userEvent.click(page.getByRole("option", { name: "Built-in client on home" }));
  await userEvent.click(sheet.getByRole("button", { name: "Grab 2" }));
  await expect.poll(() => server.grabbed).toHaveLength(1);
  expect(server.grabbed[0]?.map(({ client }) => client)).toEqual([
    { builtInOn: seedboxCard.nodeId },
    { builtInOn: homeCard.nodeId },
  ]);
});

test("after Grab each row says what its pick did, and a held cell offers to cancel that Grab and grab this one", async () => {
  const held = "A Grab is already downloading 1080p for E02. Cancel it to grab this one.";
  const server = manualServer({
    plan: twoGrabs,
    results: (picks) =>
      picks.map((pick) =>
        pick.cancelGrabId !== undefined || pick.releaseId.endsWith(".1")
          ? { grabId: heldGrabId, problem: null }
          : { grabId: null, problem: { code: "cell_held", detail: held, grabId: heldGrabId } },
      ),
  });
  openSheet(server);
  const sheet = sheetNamed("Season 1 of The Long Shelf");
  await userEvent.click(sheet.getByRole("button", { name: "Grab 2" }));
  const rows = sheet.getByRole("list", { name: "Grabs" }).getByRole("listitem");
  await expect.element(rows.first().getByText("Grabbed.")).toBeVisible();
  await expect.element(rows.nth(1).getByRole("alert")).toHaveTextContent(held);
  await expect.element(sheet.getByRole("button", { name: "Grab 1" })).toBeVisible();

  await userEvent.click(sheet.getByRole("button", { name: "Cancel it and grab this one" }));
  await expect.element(rows.nth(1).getByText("Grabbed.")).toBeVisible();
  expect(server.grabbed.at(-1)).toEqual([
    { releaseId: `${runId}.2`, cells: ["cell-e02"], lifts: [], cancelGrabId: heldGrabId },
  ]);
  await expect.element(sheet.getByRole("button", { name: "Grab 0" })).toBeDisabled();
});

test("a grabbed row that a later plan fills with another Release keeps its result, and the new Release shows none and is ticked", async () => {
  const server = manualServer({ plan: twoGrabs });
  openSheet(server);
  const sheet = sheetNamed("Season 1 of The Long Shelf");
  await userEvent.click(sheet.getByRole("button", { name: "Grab 2" }));
  const rows = sheet.getByRole("list", { name: "Grabs" }).getByRole("listitem");
  await expect.element(rows.nth(1).getByText("Grabbed.")).toBeVisible();

  await userEvent.click(rows.nth(1).getByRole("button", { name: "Other choices" }));
  await userEvent.click(rows.nth(1).getByRole("button", { name: "Use this" }));
  await expect.element(sheet.getByRole("checkbox", { name: "WEBRip-1080p · RARBG" })).toBeChecked();
  await expect.element(rows.nth(1).getByText("Grabbed.")).toBeVisible();
  await expect.element(rows.nth(2).getByText("Grabbed.")).not.toBeInTheDocument();
  await expect.element(sheet.getByRole("button", { name: "Grab 1" })).toBeEnabled();
});

test("Use this swaps the row's Release and leaves its Other choices open", async () => {
  openSheet(manualServer({ plan: twoGrabs }));
  const sheet = sheetNamed("Season 1 of The Long Shelf");
  const row = sheet.getByRole("list", { name: "Grabs" }).getByRole("listitem").nth(1);
  await userEvent.click(row.getByRole("button", { name: "Other choices" }));
  await userEvent.click(row.getByRole("button", { name: "Use this" }));
  await expect.element(sheet.getByRole("checkbox", { name: "WEBRip-1080p · RARBG" })).toBeChecked();
  await expect.element(row.getByRole("list", { name: "Other choices" })).toBeVisible();
});

test("each grabbed row still says Grabbed. after the plan read again leaves its cell out as held", async () => {
  let grabbed = false;
  const server = manualServer({
    plan: (pins) => {
      const plan = twoGrabs(pins);
      return grabbed
        ? {
            ...plan,
            plan: {
              headline: "Every Target here has its file or a Grab downloading one.",
              held: plan.plan.rows.map((row, at) => ({
                grabId: `0199f3a1-5c2e-7d4f-8a6b-${String(at).padStart(12, "0")}`,
                stage: "downloading" as const,
                words: `A Grab is downloading ${row.pick?.release.words ?? ""} for 1080p.`,
              })),
              rows: [],
            },
          }
        : plan;
    },
  });
  const live = openSheet(server);
  const sheet = sheetNamed("Season 1 of The Long Shelf");
  await userEvent.click(sheet.getByRole("button", { name: "Grab 2" }));
  await expect.poll(() => server.grabbed).toHaveLength(1);
  grabbed = true;
  live.clients[0]?.send({
    event: "grabs.search-changed",
    data: { workId, manualSearchId, searchId: runId },
  });
  await expect
    .element(sheet.getByText("Every Target here has its file or a Grab downloading one."))
    .toBeVisible();
  const rows = sheet.getByRole("list", { name: "Grabs" }).getByRole("listitem");
  await expect.element(rows.first().getByText("WEBDL-1080p · NTb")).toBeVisible();
  await expect.element(rows.first().getByText("Grabbed.")).toBeVisible();
  await expect.element(rows.nth(1).getByText("WEBDL-1080p · FLUX")).toBeVisible();
  await expect.element(rows.nth(1).getByText("Grabbed.")).toBeVisible();
  await expect
    .element(sheet.getByRole("list", { name: "Cells a Grab is downloading" }))
    .not.toBeInTheDocument();
  await expect.element(sheet.getByRole("button", { name: "Grab 0" })).toBeDisabled();
});

const pickingBlocked = (row: GrabsPlanRow): GrabsPlanRow =>
  row.pick === null ? row : { ...row, pick: { ...row.pick, release: blocked, lifts: [] } };

test("Other choices reads a blocked Release's since and why, and a lift as an amber override; Remove from Blocklist and grab posts unblock", async () => {
  const server = manualServer({
    plan: (pins) =>
      pins.includes(blockedChoice.pin)
        ? {
            ...pinnedFilmPlan,
            plan: {
              ...pinnedFilmPlan.plan,
              rows: pinnedFilmPlan.plan.rows.map(pickingBlocked),
            },
            pins: [{ key: "cell-orchard-1080p", pin: blockedChoice.pin }],
          }
        : filmPlan,
  });
  openSheet(server);
  const sheet = sheetNamed("The Quiet Orchard (2021)");
  await userEvent.click(sheet.getByRole("button", { name: "Other choices" }));
  const choices = sheet.getByRole("list", { name: "Other choices" });
  await expect
    .element(choices.getByText("From TorrentLeech, which automatic search doesn't use."))
    .toBeVisible();
  const lifted = choices.getByRole("listitem").first();
  await expect.element(lifted.getByText("Override")).toBeVisible();
  await expect
    .element(lifted.getByText("Its score of 5 is under 1080p's minimum of 10."))
    .toBeVisible();
  await expect.element(choices.getByText(blocked.blocked ?? "")).toBeVisible();

  await userEvent.click(choices.getByRole("button", { name: "Remove from Blocklist and grab" }));
  await expect
    .element(sheet.getByRole("checkbox", { name: "Bluray-1080p · SPARKS · 9.8 GB" }))
    .toBeChecked();
  await userEvent.click(sheet.getByRole("button", { name: "Grab 1" }));
  await expect
    .poll(() => server.grabbed)
    .toEqual([
      [{ releaseId: blocked.id, cells: ["cell-orchard-1080p"], lifts: [], unblock: true }],
    ]);
});

const editionWords =
  "Its name gives the Extended Edition, which the Work doesn't want. It's grabbed as the Standard Edition.";

const asStandard = (row: GrabsPlanRow): GrabsPlanRow =>
  row.pick === null
    ? row
    : { ...row, pick: { ...row.pick, edition: { edition: "standard", words: editionWords } } };

test("a pick grabbed as another Edition says so as an amber override, and Grab posts that Edition", async () => {
  const server = manualServer({
    plan: () => ({
      ...filmPlan,
      plan: { ...filmPlan.plan, rows: filmPlan.plan.rows.map(asStandard) },
    }),
  });
  openSheet(server);
  const sheet = sheetNamed("The Quiet Orchard (2021)");
  const row = sheet.getByRole("list", { name: "Grabs" }).getByRole("listitem").first();
  await expect.element(row.getByText("Override")).toBeVisible();
  await expect.element(row.getByText(editionWords)).toBeVisible();

  await userEvent.click(sheet.getByRole("button", { name: "Grab 1" }));
  await expect
    .poll(() => server.grabbed)
    .toEqual([
      [{ releaseId: flux.id, cells: ["cell-orchard-1080p"], lifts: [], edition: "standard" }],
    ]);
});
