import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import {
  downloadClients,
  homeCard,
  seedboxCard,
  thresholds,
  unbound,
} from "../test/download-clients";
import { seedboxId } from "../test/nodes";
import { renderApp } from "../test/render-app";

const path = "/settings/download-clients";
const card = (node: string) => page.getByRole("article", { name: `Built-in client on ${node}` });

test("each downloading Node's card leads with its port verdict and links to its Torrent client", async () => {
  const notListening = {
    ...seedboxCard,
    client: { ...unbound, listen: "0.0.0.0:51413", listenError: "address already in use" },
  };
  const { router } = renderApp({
    path,
    handlers: downloadClients({ cards: [homeCard, notListening], thresholds }).handlers,
  });
  await expect
    .element(
      card("home").getByText(
        "Listening on 192.168.1.20:6881. No peer from the internet has connected in yet, so port 6881 is unproven.",
      ),
    )
    .toBeVisible();
  await expect
    .element(card("home").getByText("12 torrents, 3.0 TB uploaded and 900 GB downloaded."))
    .toBeVisible();
  await expect.element(card("seedbox").getByText("Not listening")).toBeVisible();
  await expect
    .element(card("seedbox").getByText("Can't listen on 0.0.0.0:51413: address already in use."))
    .toBeVisible();
  await expect
    .element(card("home").getByText("Goes away when this Node stops downloading."))
    .toBeVisible();
  expect(page.getByRole("button", { name: /remove/iu }).query()).toBeNull();

  await userEvent.click(card("home").getByRole("button", { name: "Details" }));
  await expect.element(card("home").getByText("-SS0400-")).toBeVisible();
  await expect.element(card("home").getByText("SlipStream/0.4.0 libtorrent/2.0.11")).toBeVisible();
  await expect.element(card("home").getByText("4 MB/s")).toBeVisible();
  await expect
    .element(card("home").getByText(/the address this computer reaches the internet from/u))
    .toBeVisible();

  await userEvent.click(
    card("seedbox").getByRole("link", { name: "Torrent client settings on seedbox" }),
  );
  await expect
    .poll(() => router.state.location.pathname)
    .toBe(`/nodes/${seedboxId}/torrent-client`);
});

test("a reachable port says peers from the internet reached it", async () => {
  renderApp({ path, handlers: downloadClients({ cards: [seedboxCard], thresholds }).handlers });
  await expect.element(card("seedbox").getByText("Reachable")).toBeVisible();
  await expect
    .element(
      card("seedbox").getByText(
        "Listening on 203.0.113.9:51413. Peers from the internet have reached port 51413.",
      ),
    )
    .toBeVisible();
});

test("a card whose disk guard holds leads with new downloads waiting for disk space", async () => {
  const gb = 1024 ** 3;
  const guarded = {
    ...seedboxCard,
    diskGuard: {
      sentence: "New downloads on seedbox wait for disk space.",
      hint: "/data has 12 GB left, below its error level of 20 GB. Free up space there, or move the downloads folder to a volume with more room. Downloads already running carry on.",
      volume: "/data",
      free: 12 * gb,
      errorBelow: 20 * gb,
      sampledAt: "2026-10-23T09:05:00Z",
    },
  };
  renderApp({ path, handlers: downloadClients({ cards: [guarded], thresholds }).handlers });
  await expect.element(card("seedbox").getByText("Waiting for disk")).toBeVisible();
  await expect
    .element(card("seedbox").getByText("New downloads on seedbox wait for disk space."))
    .toBeVisible();
  await expect
    .element(
      card("seedbox").getByText(
        "/data has 12 GB left, below its error level of 20 GB. Free up space there, or move the downloads folder to a volume with more room. Downloads already running carry on.",
      ),
    )
    .toBeVisible();
  await expect.element(card("seedbox").getByText("Reachable")).toBeVisible();
});

test("when a release counts as bad saves each threshold on its own, in seconds", async () => {
  const clients = downloadClients();
  renderApp({ path, handlers: clients.handlers });
  await expect
    .element(page.getByRole("heading", { name: "When a release counts as bad" }))
    .toBeVisible();
  await userEvent.fill(
    page.getByRole("textbox", { name: "Unregistered twice, this far apart" }),
    "45",
  );
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Download Clients settings saved")).toBeVisible();
  expect(clients.writes).toEqual([{ path: "thresholds", body: { unregisteredGap: 2700 } }]);
});

test("with no Node downloading, the section says how to get a client", async () => {
  renderApp({ path, handlers: downloadClients({ cards: [], thresholds }).handlers });
  await expect
    .element(page.getByText(/^No Node downloads yet\. Give one Downloading in its/u))
    .toBeVisible();
});
