import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { longShelf } from "../../test/anthology";
import { seasonPlan } from "../../test/manual";
import { manualEntry, manualSheet } from "../../test/manual-entry";
import { renderApp } from "../../test/render-app";
import { shelfCells } from "../../test/shelf-cells";
import { starter } from "../../test/targets";
import { asPhone } from "../../test/viewport";
import { workServer } from "../../test/work-server";

// Each test opens its own Work, since a page remembers its open levels for the session.
function open() {
  const work = { ...longShelf, id: crypto.randomUUID() };
  const manual = manualEntry(seasonPlan);
  const server = workServer({ work, startCells: shelfCells(work, () => "missing") });
  renderApp({ path: `/works/${work.id}`, handlers: [...manual.handlers, ...server.handlers] });
  return { manual };
}

const levels = () => page.getByRole("region", { name: "Volumes" });
const sheet = () => manualSheet(seasonPlan);
const [volumeOne] = longShelf.levels;
const [, second] = longShelf.items;

test("a level's menu opens Manual search on the level, and its Search stays one press", async () => {
  const { manual } = open();
  await expect.element(levels().getByRole("button", { name: "Search Volume 1" })).toBeVisible();
  await userEvent.click(levels().getByRole("button", { name: "More actions for Volume 1" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Manual search" }));
  await expect.element(sheet()).toBeVisible();
  await expect.poll(() => manual.started).toEqual([{ levelIds: [volumeOne?.id] }]);
});

test("an Item row's menu holds Details and Manual search, which opens on the Item", async () => {
  const { manual } = open();
  await expect.element(levels().getByRole("button", { name: "Search V01S02" })).toBeVisible();
  await userEvent.click(levels().getByRole("button", { name: "More actions for V01S02" }));
  const menu = page.getByRole("menu", { name: "Actions for V01S02" });
  expect(
    menu
      .getByRole("menuitem")
      .elements()
      .map((item) => item.textContent),
  ).toEqual(["Details", "Manual search"]);
  await userEvent.click(menu.getByRole("menuitem", { name: "Manual search" }));
  await expect.element(sheet()).toBeVisible();
  await expect.poll(() => manual.started).toEqual([{ itemIds: [second?.id] }]);
});

test("a square's popover opens Manual search on its cell and closes", async () => {
  const { manual } = open();
  await userEvent.click(levels().getByRole("button", { name: "V01S02 1080p, Missing" }));
  const popover = page.getByRole("dialog", { name: "V01S02 1080p" });
  await userEvent.click(popover.getByRole("button", { name: "Manual search" }));
  await expect.element(sheet()).toBeVisible();
  await expect.element(popover).not.toBeInTheDocument();
  await expect
    .poll(() => manual.started)
    .toEqual([{ cell: { itemId: second?.id, edition: "any", targetId: starter.id } }]);
});

test("on a phone an Item's sheet gives way to Manual search on the Item", async () => {
  await asPhone();
  const { manual } = open();
  await userEvent.click(levels().getByRole("button", { name: /^V01S02 ?Story V01S02/u }));
  const itemSheet = page.getByRole("dialog", { name: "V01S02 Story V01S02" });
  await expect
    .element(itemSheet.getByRole("button", { name: "Search", exact: true }))
    .toBeVisible();
  await userEvent.click(itemSheet.getByRole("button", { name: "Manual search" }));
  await expect.element(sheet()).toBeVisible();
  await expect.poll(() => page.getByRole("dialog").elements()).toHaveLength(1);
  await expect.poll(() => manual.started).toEqual([{ itemIds: [second?.id] }]);
});

test("on a phone a level's actions hold Manual search", async () => {
  await asPhone();
  const { manual } = open();
  await userEvent.click(levels().getByRole("button", { name: "More actions for Volume 1" }));
  await userEvent.click(
    page
      .getByRole("dialog", { name: "Actions for Volume 1" })
      .getByRole("button", { name: "Manual search" }),
  );
  await expect.element(sheet()).toBeVisible();
  await expect.poll(() => page.getByRole("dialog").elements()).toHaveLength(1);
  await expect.poll(() => manual.started).toEqual([{ levelIds: [volumeOne?.id] }]);
});
