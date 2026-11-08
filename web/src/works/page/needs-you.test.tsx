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

test("a download waiting on a client that isn't responding names the client and offers another", async () => {
  const inTransmission = {
    ...stuck,
    downloadClient: {
      id: "7d3f9b1e-5a2c-4e8f-b6d0-1c3e5a7f9b2d",
      kind: "transmission",
      name: "Transmission",
    },
  } as const;
  const needsYou = openWith(
    [{ kind: "waiting", grabId: inTransmission.id, code: "client_away" }],
    [inTransmission],
  );
  await expect
    .element(needsYou)
    .toHaveTextContent(
      "Needs you1WaitingWaiting for Transmission on seedboxIts download carries on once Transmission responds again.Send to another client",
    );
});

test("a download waiting to fetch for an external client on an offline Node names the client", async () => {
  const toQbittorrent = {
    ...stuck,
    stage: "fetching",
    torrentId: null,
    waitsFor: {
      nodeId: stuck.nodeId,
      client: {
        id: "2b4d6f8a-0c1e-4a3b-9d5f-7e9a1c3b5d7f",
        kind: "qbittorrent",
        name: "qBittorrent",
      },
    },
  } as const;
  const needsYou = openWith(
    [{ kind: "waiting", grabId: toQbittorrent.id, code: "node_away" }],
    [toQbittorrent],
  );
  await expect
    .element(needsYou)
    .toHaveTextContent(
      "Needs you1WaitingWaiting to send to qBittorrent on seedboxseedbox is offline. This download goes there once seedbox is back online.Send to another client",
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
