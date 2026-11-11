import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { FilesCellList, GrabsGrab } from "../../api/gen/model";
import { imported } from "../../test/grabs";
import { orchardId, orchardItemId } from "../../test/library";
import { filmPlan } from "../../test/manual";
import { manualEntry, manualSheet } from "../../test/manual-entry";
import { renderApp } from "../../test/render-app";
import { keeper, starter } from "../../test/targets";
import { asPhone } from "../../test/viewport";
// oxlint-disable-next-line import/max-dependencies -- the entry points sit on cells, Needs you and the hero
import { failedCells, filled, upgradable } from "../../test/work-cells";
import { workServer } from "../../test/work-server";

function open(startCells?: FilesCellList) {
  const manual = manualEntry(filmPlan);
  const work = workServer(startCells === undefined ? {} : { startCells });
  renderApp({ path: `/works/${orchardId}`, handlers: [...manual.handlers, ...work.handlers] });
  return { manual, work };
}

const cellOf = (targetId: string) => ({
  cell: { itemId: orchardItemId, edition: "any", targetId },
});
const sheet = () => manualSheet(filmPlan);
const main = () => page.getByRole("main");

test("the hero's Manual search opens the sheet on the whole Work", async () => {
  const { manual } = open();
  await userEvent.click(main().getByRole("button", { name: "Manual search", exact: true }));
  await expect.element(sheet()).toBeVisible();
  await expect.poll(() => manual.started).toEqual([{}]);
});

test("a missing cell's Manual search plans for that cell alone", async () => {
  const { manual } = open();
  await expect
    .element(main().getByRole("button", { name: "Search for 1080p, Any cut" }))
    .toBeVisible();
  await userEvent.click(main().getByRole("button", { name: "Manual search for 1080p, Any cut" }));
  await expect.element(sheet()).toBeVisible();
  await expect.poll(() => manual.started).toEqual([cellOf(starter.id)]);
});

test("a filled cell has Manual search beside it and in its Details, and Details closes as the sheet opens", async () => {
  const { manual } = open(filled);
  await expect
    .element(main().getByRole("button", { name: "Manual search for 1080p, Any cut" }))
    .toBeVisible();
  expect(
    main().getByRole("button", { name: "Search for 1080p, Any cut", exact: true }).elements(),
  ).toHaveLength(0);

  await userEvent.click(main().getByRole("button", { name: "Details of 1080p" }));
  const details = page.getByRole("dialog", { name: "Details of 1080p" });
  await userEvent.click(details.getByRole("button", { name: "Manual search" }));
  await expect.element(sheet()).toBeVisible();
  await expect.element(details).not.toBeInTheDocument();
  await expect.poll(() => manual.started).toEqual([cellOf(starter.id)]);
});

const failedGrab: GrabsGrab = {
  ...imported,
  stage: "failed",
  history: [...imported.history.slice(0, -1), { stage: "failed", at: "2026-10-12T09:03:02Z" }],
  failure: {
    stage: "importing",
    code: "import_failed",
    message: "place movies/The Quiet Orchard (2021): permission denied",
    needsYou: true,
    current: true,
  },
};

test("a failed Grab under Needs you opens Manual search on its Item beside Search again", async () => {
  const { manual, work } = open();
  await expect
    .element(main().getByRole("button", { name: "Manual search", exact: true }))
    .toBeVisible();
  work.grabMoves(failedGrab, failedCells(failedGrab.id));

  const card = page
    .getByRole("region", { name: "Needs you" })
    .getByRole("article", { name: "2160p HDR failed at Import · 1080p waits on it" });
  await expect.element(card.getByRole("button", { name: "Search again" })).toBeVisible();
  await userEvent.click(card.getByRole("button", { name: "Manual search" }));
  await expect.element(sheet()).toBeVisible();
  await expect.poll(() => manual.started).toEqual([{ itemIds: [orchardItemId] }]);
});

test("an upgrade under Needs you opens Manual search on its Items", async () => {
  const { manual } = open({
    ...upgradable,
    needsYou: [
      { kind: "upgrade", itemIds: [orchardItemId], edition: "any", upgrades: [starter.id] },
    ],
  });
  const upgrades = page.getByRole("list", { name: "Upgrades available" });
  await userEvent.click(upgrades.getByRole("button", { name: "Manual search" }));
  await expect.element(sheet()).toBeVisible();
  await expect.poll(() => manual.started).toEqual([{ itemIds: [orchardItemId] }]);
});

test("on a phone the hero's actions hold Manual search, and the sheet is the only one open", async () => {
  await asPhone();
  const { manual } = open();
  await userEvent.click(main().getByRole("button", { name: "More actions for The Quiet Orchard" }));
  await userEvent.click(
    page
      .getByRole("dialog", { name: "Actions for The Quiet Orchard" })
      .getByRole("button", { name: "Manual search" }),
  );
  await expect.element(sheet()).toBeVisible();
  await expect.poll(() => page.getByRole("dialog").elements()).toHaveLength(1);
  await expect.poll(() => manual.started).toEqual([{}]);
});

test("on a phone a cell's action opens a menu with Search and Manual search", async () => {
  await asPhone();
  const { manual } = open();
  await userEvent.click(main().getByRole("button", { name: "Search for 2160p HDR, Any cut" }));
  const actions = page.getByRole("dialog", { name: "Actions for 2160p HDR, Any cut" });
  await expect.element(actions.getByRole("button", { name: "Search", exact: true })).toBeVisible();
  await userEvent.click(actions.getByRole("button", { name: "Manual search" }));
  await expect.element(sheet()).toBeVisible();
  await expect.poll(() => page.getByRole("dialog").elements()).toHaveLength(1);
  await expect.poll(() => manual.started).toEqual([cellOf(keeper.id)]);
});

test("on a phone a filled cell's menu holds Manual search alone", async () => {
  await asPhone();
  const { manual } = open(filled);
  await userEvent.click(main().getByRole("button", { name: "Search for 1080p, Any cut" }));
  const actions = page.getByRole("dialog", { name: "Actions for 1080p, Any cut" });
  await expect.element(actions.getByRole("button", { name: "Manual search" })).toBeVisible();
  expect(actions.getByRole("button", { name: "Search", exact: true }).elements()).toHaveLength(0);
  await userEvent.click(actions.getByRole("button", { name: "Manual search" }));
  await expect.element(sheet()).toBeVisible();
  await expect.poll(() => manual.started).toEqual([cellOf(starter.id)]);
});
