import { HttpResponse } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { downloadClients, usableFolders, withTorrent } from "../../../test/download-clients";
import { nodeSystems } from "../../../test/node-system";
import { home, homeId, nodesAre, seedbox, seedboxId } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";
import { asPhone } from "../../../test/viewport";

const path = `/nodes/${homeId}/torrent-client`;

function serve({ config = withTorrent() } = {}) {
  const clients = downloadClients();
  const systems = nodeSystems({ configs: { [homeId]: config } });
  const handlers = [...clients.handlers, ...systems.handlers, ...usableFolders];
  return { clients, systems, handlers };
}

const tabs = () => page.getByRole("navigation", { name: "home" });

test("Torrent client leads with the port verdict, then its restart rows and live limits", async () => {
  renderApp({ path, handlers: serve().handlers });
  await expect
    .element(
      page.getByText(
        "Listening on 192.168.1.20:6881. No peer from the internet has connected to port 6881 yet, so it may not be reachable.",
      ),
    )
    .toBeVisible();
  await expect
    .element(tabs().getByRole("link", { name: "Torrent client" }))
    .toHaveAttribute("aria-current", "page");
  await expect
    .element(page.getByRole("textbox", { name: "Downloads folder" }))
    .toHaveValue("/data/home/downloads");
  await expect
    .element(
      page.getByText(
        "With 0.0.0.0 or [::], the client listens on the address this computer reaches the internet from.",
      ),
    )
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await expect
    .element(page.getByText(/^Trackers reachable only through another network adapter/u))
    .toBeVisible();
  await Promise.all(
    ["Speed", "Queue", "Peers"].map((name) =>
      expect.element(page.getByRole("heading", { name })).toBeVisible(),
    ),
  );
  await expect
    .element(page.getByRole("switch", { name: "uTP, incoming and outgoing" }))
    .toBeChecked();
  await expect
    .element(page.getByRole("main").getByRole("link", { name: "Torrents" }))
    .toHaveAttribute("href", `/activity/torrents?node=${homeId}&tab=all`);
});

test("a limit saves on its own, as the one field it changed", async () => {
  const { clients, handlers } = serve();
  renderApp({ path, handlers });
  await userEvent.fill(page.getByRole("textbox", { name: "Downloads at once" }), "5");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Torrent client settings saved")).toBeVisible();
  await userEvent.click(page.getByRole("switch", { name: "uTP, incoming and outgoing" }));
  await expect
    .poll(() => clients.writes)
    .toEqual([
      { path: `${homeId}/settings`, body: { activeDownloads: 5 } },
      { path: `${homeId}/settings`, body: { utp: false } },
    ]);
});

test("a downloads folder the Node can't use shows the Node's reason under the field", async () => {
  const { systems, handlers } = serve();
  systems.state.refuse = HttpResponse.json(
    {
      status: 422,
      code: "folder_unusable",
      title: "SlipStream can't use this folder",
      detail: "Can't use this folder because it isn't writable.",
      errors: [{ location: "body.torrent.downloads", message: "isn't writable" }],
    },
    { status: 422, headers: { "Content-Type": "application/problem+json" } },
  );
  renderApp({ path, handlers });
  const folder = page.getByRole("textbox", { name: "Downloads folder" });
  await userEvent.clear(folder);
  await userEvent.type(folder, "/srv/readonly");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect
    .element(page.getByRole("alert"))
    .toHaveTextContent("Can't use this folder because it isn't writable.");
  expect(systems.writes).toEqual([
    { method: "PATCH", nodeId: homeId, body: { "torrent.downloads": "/srv/readonly" } },
  ]);
});

test("a saved listen address waits for a restart and says so", async () => {
  const pending = withTorrent({
    listen: { value: "0.0.0.0:51413", isDefault: false, pendingRestart: true },
  });
  renderApp({ path, handlers: serve({ config: pending }).handlers });
  await expect.element(page.getByText("Restart home to use the new listen address")).toBeVisible();
  await expect.element(page.getByText("It listens on 0.0.0.0:6881 until then.")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Restart now" })).toBeVisible();
});

test("an offline Node keeps its live limits and says its restart rows wait for it", async () => {
  const away = { ...seedbox, roles: [{ role: "downloading", state: "away" } as const] };
  const { handlers } = serve();
  renderApp({
    path: `/nodes/${seedboxId}/torrent-client`,
    handlers: [...handlers, nodesAre(home, away)],
  });
  await expect
    .element(
      page.getByText(
        "seedbox is offline. Its listen address and downloads folder show here when it reconnects.",
      ),
    )
    .toBeVisible();
  await expect.element(page.getByRole("textbox", { name: "Downloads at once" })).toHaveValue("3");
});

test("a Node without Downloading has no Torrent client tab, and its address is not found", async () => {
  const searching = { ...home, roles: [{ role: "searching", state: "holding" } as const] };
  const { handlers } = serve();
  renderApp({ path: `/nodes/${homeId}/web`, handlers: [...handlers, nodesAre(searching)] });
  await expect.element(tabs().getByRole("link", { name: "Web and API" })).toBeVisible();
  expect(tabs().getByRole("link", { name: "Torrent client" }).query()).toBeNull();

  renderApp({ path, handlers: [...handlers, nodesAre(searching)] });
  await expect.element(page.getByText("There's nothing at this address.")).toBeVisible();
});

test("on a phone the Node page lists Torrent client as a pushed section", async () => {
  await asPhone();
  const { router } = renderApp({ path: `/nodes/${homeId}`, handlers: serve().handlers });
  await userEvent.click(page.getByRole("link", { name: "Torrent client" }));
  await expect.poll(() => router.state.location.pathname).toBe(path);
  await expect.element(page.getByRole("textbox", { name: "Downloads at once" })).toHaveValue("3");
});
