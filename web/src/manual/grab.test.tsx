import { expect, test } from "vitest";
import { userEvent } from "vitest/browser";

import { heldGrabId, manualSearchId, runId } from "../test/manual";
import { manualServer } from "../test/manual-server";
import { openSheet, sheetNamed, twoGrabs, workId } from "../test/manual-sheet";

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
  await expect
    .element(sheet.getByRole("checkbox", { name: "WEBRip-1080p · RARBG · 4.0 GB" }))
    .toBeChecked();
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
  await expect
    .element(sheet.getByRole("checkbox", { name: "WEBRip-1080p · RARBG · 4.0 GB" }))
    .toBeChecked();
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
                words: `A Grab is downloading ${row.pick?.release.title ?? ""} for 1080p.`,
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
  await expect.element(rows.first().getByText("WEBDL-1080p · NTb · 4.0 GB")).toBeVisible();
  await expect.element(rows.first().getByText("Grabbed.")).toBeVisible();
  await expect.element(rows.nth(1).getByText("WEBDL-1080p · FLUX · 4.0 GB")).toBeVisible();
  await expect.element(rows.nth(1).getByText("Grabbed.")).toBeVisible();
  await expect
    .element(sheet.getByRole("list", { name: "Files already downloading" }))
    .not.toBeInTheDocument();
  await expect.element(sheet.getByRole("button", { name: "Grab 0" })).toBeDisabled();
});
