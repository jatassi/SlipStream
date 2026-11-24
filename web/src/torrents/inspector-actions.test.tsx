import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { MovesMoveAsk } from "../api/gen/model";
import { downloadClients, seedboxCard } from "../test/download-clients";
import { liveServer } from "../test/live-server";
import { seedboxId } from "../test/nodes";
import { renderApp } from "../test/render-app";
import { actionsServer, detailOf, torrent, torrentsServer } from "../test/torrents";

const finished = torrent({
  state: "seeding",
  tabs: ["seeding", "all"],
  name: "Orchard.Harvest.2021.1080p",
  size: 1000,
  done: 1000,
});

function openMove(answer: (ask: MovesMoveAsk) => Response) {
  const previews: MovesMoveAsk[] = [];
  const actions = actionsServer();
  const clients = downloadClients();
  clients.state.list = { ...clients.state.list, cards: [...clients.state.list.cards, seedboxCard] };
  renderApp({
    path: `/activity/torrents/${finished.id}?tab=all`,
    handlers: [
      actions.handler,
      ...clients.handlers,
      ...liveServer().handlers,
      ...torrentsServer([finished], (row) => detailOf(row)).handlers,
      http.post("*/api/v1/moves/preview", async ({ request }) => {
        // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the generated client sends the body it types
        const ask = (await request.json()) as MovesMoveAsk;
        previews.push(ask);
        return answer(ask);
      }),
    ],
  });
  return { previews, actions };
}

async function chooseMove() {
  const inspector = page.getByRole("complementary", { name: "Orchard.Harvest.2021.1080p" });
  await userEvent.click(
    inspector.getByRole("button", { name: "More actions for Orchard.Harvest.2021.1080p" }),
  );
  await userEvent.click(page.getByRole("menuitem", { name: "Move to…" }));
  return page.getByRole("dialog", { name: "Move Orchard.Harvest.2021.1080p" });
}

test("Move to… in the Inspector opens the Move Dialog and never acts on the torrent", async () => {
  const { previews, actions } = openMove(() =>
    HttpResponse.json({ status: 500, code: "unexpected", title: "Held" }, { status: 500 }),
  );
  await expect.element(await chooseMove()).toBeVisible();
  await expect
    .poll(() => previews)
    .toEqual([
      { origin: "torrent", works: [], torrents: [{ torrentId: finished.id, nodeId: seedboxId }] },
    ]);
  expect(actions.bodies).toEqual([]);
});
