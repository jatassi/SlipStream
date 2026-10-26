import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { GrabsGrab } from "../../../api/gen/model";
import { getListGrabsMockHandler } from "../../../api/gen/query/grabs/grabs.msw";
import { imported } from "../../../test/grabs";
import { renderApp } from "../../../test/render-app";

const searching = (peers: number): GrabsGrab => ({
  ...imported,
  stage: "fetching",
  torrentId: null,
  infoHash: null,
  history: [{ stage: "fetching", at: imported.createdAt }],
  metadata: { peers, endsAt: new Date(Date.now() + 30 * 60_000).toISOString() },
});

const row = () => page.getByRole("row", { name: new RegExp(imported.releaseTitle, "u") });

test.for([
  [0, "Looking for peers"],
  [1, "Getting metadata from 1 peer"],
  [3, "Getting metadata from 3 peers"],
] as const)(
  "a Job fetching from a magnet link with %i peers shows them and when it fails without the metadata",
  async ([peers, line]) => {
    renderApp({
      path: "/activity",
      handlers: [getListGrabsMockHandler({ grabs: [searching(peers)] })],
    });
    await expect
      .element(row())
      .toMatchTextContent(
        new RegExp(`Fetching${line}Fails at .+ if no peer sends the metadata$`, "u"),
      );
  },
);
