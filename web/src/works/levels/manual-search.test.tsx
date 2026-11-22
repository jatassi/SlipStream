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
