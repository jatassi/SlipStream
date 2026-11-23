import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { MovesMoveAsk, MovesMovePreview } from "../api/gen/model";
import { downloadClients, seedboxCard } from "../test/download-clients";
import { liveServer } from "../test/live-server";
import { homeId, seedboxId } from "../test/nodes";
import { renderApp } from "../test/render-app";
import { actionsServer, detailOf, torrent, torrentsServer } from "../test/torrents";

const finished = torrent({
  state: "seeding",
  tabs: ["seeding", "all"],
  name: "Orchard.Harvest.2021.1080p",
  size: 1000,
  done: 1000,
});

const windowStarts: MovesMovePreview = {
  token: "torrent-1",
  expiresAt: "2099-01-01T00:00:00Z",
  recommendation: "Move 1 torrent to home.",
  button: "Move 1",
  movable: 1,
  total: 1,
  homed: 0,
  units: [
    {
      key: `torrent:${finished.id}`,
      kind: "torrent",
      name: finished.name,
      workId: "00000000-0000-0000-0000-000000000000",
      torrentId: finished.id,
      from: { nodeId: seedboxId, nodeName: "seedbox", root: "", rootId: "", client: "Built-in" },
      to: { nodeId: homeId, nodeName: "home", root: "", rootId: "", client: "Built-in" },
      verdict: "amber",
      bytes: 1000,
      reasons: [
        {
          check: "hit_and_run",
          code: "window_starts",
          color: "amber",
          words: "SlipStream starts counting this Hit and run window from now.",
        },
      ],
    },
  ],
};

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

test("the torrent Move Dialog shows the Hit and run note a Move brings", async () => {
  openMove(() => HttpResponse.json(windowStarts));
  const notes = (await chooseMove()).getByRole("region", { name: "Moves, with a note" });
  await expect
    .element(notes.getByRole("listitem"))
    .toHaveTextContent(
      "Orchard.Harvest.2021.1080p SlipStream starts counting this Hit and run window from now.",
    );
});
