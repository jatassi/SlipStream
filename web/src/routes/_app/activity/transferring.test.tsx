import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { GrabsGrab } from "../../../api/gen/model";
import { getListGrabsMockHandler } from "../../../api/gen/query/grabs/grabs.msw";
import { getListWorksMockHandler } from "../../../api/gen/query/works/works.msw";
import { downloading } from "../../../test/grabs";
import { orchardListed } from "../../../test/library";
import { liveServer } from "../../../test/live-server";
import { renderApp } from "../../../test/render-app";
import { starter, targetHandlers } from "../../../test/targets";
// oxlint-disable-next-line import/max-dependencies -- the Jobs table reads Grabs, Works, Targets, Torrents, Transfers and the live stream
import { torrentsServer } from "../../../test/torrents";
import { copying, transfersApi, waiting } from "../../../test/transfers";

const transferring: GrabsGrab = {
  ...downloading,
  stage: "transferring",
  history: [...downloading.history, { stage: "transferring", at: "2026-10-12T09:02:00Z" }],
};

const quietOrchard = () => page.getByRole("row", { name: /^The Quiet Orchard /u });

test("a transferring Job reads its Transfer: where it copies from and how fast, or why it waits", async () => {
  const transfers = transfersApi({ transfers: [{ ...copying, job: transferring.id }] });
  const { clients, handlers: liveHandlers } = liveServer();
  renderApp({
    path: "/activity",
    handlers: [
      ...targetHandlers([{ ...starter, id: transferring.targetIds[0] ?? "" }]),
      getListWorksMockHandler({ works: [orchardListed] }),
      getListGrabsMockHandler({ grabs: [transferring] }),
      ...torrentsServer([]).handlers,
      ...transfers.handlers,
      ...liveHandlers,
    ],
  });
  await expect
    .element(quietOrchard())
    .toMatchTextContent("Transferring from seedbox25%2.1 GB of 8.4 GB · 48 MB/s · 2 min left");

  transfers.state.transfers = [
    { ...waiting, job: transferring.id, verified: copying.verified, total: copying.total },
  ];
  clients[0]?.send({ event: "transfers.changed", data: { transferIds: [], nodeIds: [] } });
  await expect
    .element(quietOrchard())
    .toMatchTextContent("Waiting for seedbox to come back.25%2.1 GB of 8.4 GB");
});
