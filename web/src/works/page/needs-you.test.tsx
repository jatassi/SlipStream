import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { FilesCellList, FilesNeed, GrabsGrab } from "../../api/gen/model";
import { getGetWorkCellsMockHandler } from "../../api/gen/query/files/files.msw";
import { getListGrabsMockHandler } from "../../api/gen/query/grabs/grabs.msw";
import { orchardId } from "../../test/library";
import { renderApp } from "../../test/render-app";
import { awayGrab as stuck, upgradable } from "../../test/work-cells";
import { workServer } from "../../test/work-server";

function openWith(needsYou: FilesNeed[], grabs: GrabsGrab[] = []) {
  const cells: FilesCellList = { ...upgradable, needsYou };
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [
      getGetWorkCellsMockHandler(cells),
      getListGrabsMockHandler({ grabs }),
      ...workServer().handlers,
    ],
  });
  return page.getByRole("region", { name: "Needs you" });
}

test("a download waiting on an away Node names it and opens it", async () => {
  const needsYou = openWith([{ kind: "waiting", grabId: stuck.id, code: "node_away" }], [stuck]);
  await expect
    .element(needsYou)
    .toHaveTextContent(
      "Needs you1WaitingWaiting for seedboxIts download carries on once seedbox is back online.Open seedbox",
    );
});

test("a Work with no library root says where to add one", async () => {
  const needsYou = openWith([{ kind: "no_root" }]);
  await expect.element(needsYou.getByText("No library root for this Work")).toBeVisible();
  await expect.element(needsYou.getByRole("button", { name: "Open Library roots" })).toBeVisible();
});

test("an upgrade names its Targets with a way to search", async () => {
  const needsYou = openWith([
    {
      kind: "upgrade",
      itemIds: [upgradable.items[0]?.itemId ?? ""],
      edition: "any",
      upgrades: [upgradable.cells[0]?.targetId ?? ""],
    },
  ]);
  const upgrades = needsYou.getByRole("list", { name: "Upgrades available" });
  await expect
    .element(upgrades)
    .toHaveTextContent("Upgrade available for 1080pA better release would replace the file.Search");
});

test("nothing needing the Admin shows no Needs you", async () => {
  openWith([]);
  await expect.element(page.getByRole("table", { name: "Editions and Targets" })).toBeVisible();
  await expect.element(page.getByRole("region", { name: "Needs you" })).not.toBeInTheDocument();
});
