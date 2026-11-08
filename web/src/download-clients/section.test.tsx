import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import {
  downloadClients,
  homeCard,
  seedboxCard,
  thresholds,
  unbound,
} from "../test/download-clients";
import { attic, atticAway, orphan, withExternals } from "../test/external-clients";
import { home, homeId, nodesAre, seedbox, seedboxId } from "../test/nodes";
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
        "Listening on 192.168.1.20:6881. No peer from the internet has connected to port 6881 yet, so it may not be reachable.",
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
    .element(card("home").getByText("This card is removed once this Node stops downloading."))
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
      hint: "/data has 12 GB left, below its error level of 20 GB. Free up space there, or move the downloads folder to a volume with more room. Downloads already running continue.",
      errorBelow: 20 * gb,
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
        "/data has 12 GB left, below its error level of 20 GB. Free up space there, or move the downloads folder to a volume with more room. Downloads already running continue.",
      ),
    )
    .toBeVisible();
  await expect.element(card("seedbox").getByText("Reachable")).toBeVisible();
});

test("when a Release counts as bad saves each threshold on its own, in seconds", async () => {
  const clients = downloadClients();
  renderApp({ path, handlers: clients.handlers });
  await expect
    .element(page.getByRole("heading", { name: "When a Release counts as bad" }))
    .toBeVisible();
  await userEvent.fill(
    page.getByRole("textbox", { name: "Unregistered twice, this far apart" }),
    "45",
  );
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Download Clients settings saved")).toBeVisible();
  expect(clients.writes).toEqual([{ path: "thresholds", body: { unregisteredGap: 2700 } }]);
});

test("the metadata wait of a magnet link saves in seconds", async () => {
  const clients = downloadClients();
  renderApp({ path, handlers: clients.handlers });
  await expect
    .element(
      page.getByText("A Grab of a magnet link fails when no peer sends its metadata in this long."),
    )
    .toBeVisible();
  await userEvent.fill(
    page.getByRole("textbox", { name: "A magnet link with no metadata fails after" }),
    "10",
  );
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Download Clients settings saved")).toBeVisible();
  expect(clients.writes).toEqual([{ path: "thresholds", body: { metadataWait: 600 } }]);
});

test("with no Node downloading, the section says how to get a client", async () => {
  renderApp({ path, handlers: downloadClients({ cards: [], thresholds }).handlers });
  await expect
    .element(page.getByText(/^No Node downloads yet\. Give one Downloading in its/u))
    .toBeVisible();
});

const externals = (node: string) => page.getByRole("list", { name: `External clients on ${node}` });

test("a Node's external clients follow its built-in client, with their status in words and a link to their web UI", async () => {
  const upstairs = {
    ...atticAway,
    id: "c0ffee00-0000-4000-8000-000000000001",
    name: "Upstairs",
    webUi: null,
  };
  renderApp({
    path,
    handlers: downloadClients({ cards: [withExternals(homeCard, attic, upstairs)] }).handlers,
  });
  const transmission = externals("home").getByRole("listitem", { name: "Transmission" });
  await expect.element(transmission.getByText("Responding")).toBeVisible();
  await expect.element(transmission.getByText("home reached Transmission 4.1.0.")).toBeVisible();
  await expect.element(transmission.getByText("Transmission, 3 torrents")).toBeVisible();
  const web = transmission.getByRole("link", { name: "Open Transmission" });
  await expect.element(web).toHaveAttribute("href", "http://192.168.1.20:9091/transmission/web/");
  await expect.element(web).toHaveAttribute("target", "_blank");

  const away = externals("home").getByRole("listitem", { name: "Upstairs" });
  await expect.element(away.getByText("Not responding")).toBeVisible();
  await expect
    .element(
      away.getByText(
        "Transmission isn't responding. Check that it's running and that home can reach 192.168.1.20:9091.",
      ),
    )
    .toBeVisible();
  expect(away.getByRole("link").query()).toBeNull();
  await expect.element(away.getByRole("button", { name: "Edit" })).toBeVisible();
  await expect.element(away.getByRole("button", { name: "Delete" })).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Add a Download Client" })).toBeVisible();
});

test("New Grabs go to offers the built-in client and each external client that is on, and saves the choice", async () => {
  const off = {
    ...attic,
    id: "c0ffee00-0000-4000-8000-000000000002",
    name: "Spare",
    on: false,
    reach: "off",
  } as const;
  const clients = downloadClients({ cards: [withExternals(homeCard, attic, off), seedboxCard] });
  renderApp({ path, handlers: clients.handlers });
  const choice = page.getByRole("combobox", { name: "New Grabs go to" });
  await expect.element(choice).toHaveTextContent("Built-in client");
  await expect
    .element(page.getByText("Grabs for Works whose library root is on home go to this client."))
    .toBeVisible();
  expect(page.getByRole("combobox").elements()).toHaveLength(1);

  await userEvent.click(choice);
  expect(
    page
      .getByRole("option")
      .elements()
      .map((option) => option.textContent),
  ).toEqual(["Built-in client", "Transmission"]);
  await userEvent.click(page.getByRole("option", { name: "Transmission" }));
  await expect.element(page.getByText("New Grabs on home go to Transmission")).toBeVisible();
  await expect.element(choice).toHaveTextContent("Transmission");
  expect(clients.writes).toEqual([{ path: `${homeId}/settings`, body: { grabsTo: attic.id } }]);
});

test("clients on Nodes that no longer download sit under their own heading with Delete only", async () => {
  renderApp({
    path,
    handlers: [
      nodesAre(home, seedbox),
      ...downloadClients({ cards: [homeCard], orphans: [orphan] }).handlers,
    ],
  });
  const section = page.getByRole("region", { name: "Nodes that no longer download" });
  await expect
    .element(
      section.getByText(
        "SlipStream doesn't check these clients, because their Nodes no longer hold Downloading. Delete each one, or give its Node Downloading again.",
      ),
    )
    .toBeVisible();
  const row = section.getByRole("listitem", { name: "qBittorrent" });
  await expect.element(row.getByText("qBittorrent on seedbox, 0 torrents")).toBeVisible();
  await expect.element(row.getByRole("button", { name: "Delete" })).toBeVisible();
  expect(row.getByRole("button", { name: "Edit" }).query()).toBeNull();
  expect(row.getByRole("link").query()).toBeNull();
});
