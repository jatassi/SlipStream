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
  pinnedFilmPlan,
  runId,
} from "../test/manual";
import { manualServer } from "../test/manual-server";
import { openSheet, sheetNamed, twoGrabs } from "../test/manual-sheet";

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

test("a grabbed row that a later plan fills with another Release shows no result and is ticked again", async () => {
  const server = manualServer({ plan: twoGrabs });
  openSheet(server);
  const sheet = sheetNamed("Season 1 of The Long Shelf");
  await userEvent.click(sheet.getByRole("button", { name: "Grab 2" }));
  const second = sheet.getByRole("list", { name: "Grabs" }).getByRole("listitem").nth(1);
  await expect.element(second.getByText("Grabbed.")).toBeVisible();

  await userEvent.click(second.getByRole("button", { name: "Other choices" }));
  await userEvent.click(second.getByRole("button", { name: "Use this" }));
  await expect.element(sheet.getByRole("checkbox", { name: "WEBRip-1080p · RARBG" })).toBeChecked();
  await expect.element(second.getByText("Grabbed.")).not.toBeInTheDocument();
  await expect.element(sheet.getByRole("button", { name: "Grab 1" })).toBeEnabled();
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

test("a pick grabbed as another Edition says so as an amber override, and Grab posts that Edition", async () => {
  const words =
    "Its name gives the Extended Edition, which the Work doesn't want. It's grabbed as the Standard Edition.";
  const server = manualServer({
    plan: () => ({
      ...filmPlan,
      plan: {
        ...filmPlan.plan,
        rows: filmPlan.plan.rows.map((row) =>
          row.pick === null
            ? row
            : { ...row, pick: { ...row.pick, edition: { edition: "standard", words } } },
        ),
      },
    }),
  });
  openSheet(server);
  const sheet = sheetNamed("The Quiet Orchard (2021)");
  const row = sheet.getByRole("list", { name: "Grabs" }).getByRole("listitem").first();
  await expect.element(row.getByText("Override")).toBeVisible();
  await expect.element(row.getByText(words)).toBeVisible();

  await userEvent.click(sheet.getByRole("button", { name: "Grab 1" }));
  await expect
    .poll(() => server.grabbed)
    .toEqual([
      [{ releaseId: flux.id, cells: ["cell-orchard-1080p"], lifts: [], edition: "standard" }],
    ]);
});
