import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { FilesCellList } from "../../api/gen/model";
import { orchardId, orchardItemId } from "../../test/library";
import { filmPlan } from "../../test/manual";
import { manualEntry, manualSheet } from "../../test/manual-entry";
import { renderApp } from "../../test/render-app";
import { keeper, starter } from "../../test/targets";
import { asPhone } from "../../test/viewport";
import { filled, upgradable } from "../../test/work-cells";
// oxlint-disable-next-line import/max-dependencies -- the entry points sit on cells, Needs you and the hero
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
