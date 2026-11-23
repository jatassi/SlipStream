import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { ids, vps } from "../../../test/landing";
import { home, nodesAre, seedbox, topologyIs, topologyOf } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";
import { board, boardIs, homeSeat, outcome, rolesServer, seedboxSeat } from "../../../test/roles";
import { copying, transfersApi } from "../../../test/transfers";

const vpsSeat = { ...seedboxSeat, id: ids.vps, name: "vps", assigned: [], roles: [] };

test("a link's Inspector lists the Moves crossing it, and the Node's Transfers name the Move", async () => {
  const moving = {
    ...copying,
    purpose: "move" as const,
    jobName: "The Quiet Orchard (2021)",
    source: { nodeId: ids.vps, nodeName: "vps" },
    destination: { ...copying.destination, nodeId: ids.home, nodeName: "home" },
  };
  renderApp({
    path: "/nodes",
    handlers: [
      nodesAre(home, seedbox, vps),
      topologyIs(topologyOf([home, seedbox, vps], { rtt: 24 })),
      boardIs(board(homeSeat, seedboxSeat, vpsSeat)),
      ...rolesServer({ plans: [outcome({})], sets: [] }).handlers,
      ...transfersApi({ transfers: [moving, copying] }).handlers,
    ],
  });
  await expect.element(page.getByRole("figure", { name: "Diagram" })).toBeVisible();
  await userEvent.click(
    page.elementLocator(document.querySelector(`[data-link="${ids.vps}"]`) ?? document.body),
  );
  const inspector = page.getByRole("complementary", { name: "home and vps" });
  await expect
    .element(inspector.getByRole("heading", { name: "Moving on this link" }))
    .toBeVisible();
  await expect
    .element(inspector.getByRole("listitem"))
    .toHaveTextContent("The Quiet Orchard (2021), from vps to home");
});
