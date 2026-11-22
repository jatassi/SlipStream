import { afterEach, expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderHome, summaryOf } from "../../test/home";
import { synced } from "../../test/live-server";
import { homeId } from "../../test/nodes";
import { resize, restoreViewport } from "../../test/viewport";
import { rateWords } from "../../torrents/model";

afterEach(restoreViewport);

const mb = 1024 * 1024;

test("the top bar and Downloading sum every Node's rates while bytes move, and the top bar's hide at 1360px", async () => {
  const { app, server } = renderHome(summaryOf());
  const downloading = page.getByRole("region", { name: "Downloading" });
  await expect.element(downloading.getByRole("list", { name: "Downloading Nodes" })).toBeVisible();
  await synced(app);
  const banner = page.getByRole("banner");
  await expect.element(banner.getByLabelText("Down")).not.toBeInTheDocument();

  server.clients[0]?.send({
    event: "nodes.traffic",
    data: {
      at: new Date().toISOString(),
      links: [],
      nodes: [
        { id: homeId, downBps: 3 * mb, upBps: mb, torrents: [] },
        { id: "5e000000-0000-4000-8000-00000000ffff", downBps: mb, upBps: 0, torrents: [] },
      ],
    },
  });
  const total = `↓ ${rateWords(4 * mb)}·↑ ${rateWords(mb)}`;
  await expect.poll(() => banner.element().textContent).toContain(total);
  await expect
    .element(downloading.getByRole("listitem"))
    .toHaveTextContent(`home·↓ ${rateWords(3 * mb)}·↑ ${rateWords(mb)}·1 downloading`);

  await resize(1360, 900);
  await expect.element(banner.getByLabelText("Down")).not.toBeInTheDocument();
  await expect.poll(() => downloading.element().textContent).toContain(total);

  server.clients[0]?.send({
    event: "nodes.traffic",
    data: {
      at: new Date().toISOString(),
      links: [],
      nodes: [{ id: homeId, downBps: 0, upBps: 0, torrents: [] }],
    },
  });
  await expect.element(downloading.getByLabelText("Down")).not.toBeInTheDocument();
  await expect.element(downloading.getByRole("listitem")).toHaveTextContent("home·1 downloading");
});
