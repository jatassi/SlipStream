import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { MediaserversServerView, NodesNode } from "../../../api/gen/model";
import { liveServer, synced } from "../../../test/live-server";
import { denJellyfin, livingRoomPlex, mediaServerHandlers } from "../../../test/media-servers";
import { home, homeId, nodesAre, seedbox, seedboxId } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";
import { asPhone } from "../../../test/viewport";

const linkHome: NodesNode = {
  ...home,
  roles: [...home.roles, { role: "media-server-link", state: "holding" }],
};
const linkSeedbox: NodesNode = {
  ...seedbox,
  standing: "in-step",
  link: { state: "up", since: "2026-10-12T10:00:00Z" },
  roles: [{ role: "media-server-link", state: "holding" }],
};
const searchSeedbox: NodesNode = {
  ...linkSeedbox,
  roles: [{ role: "searching", state: "holding" }],
};

const answering: MediaserversServerView = {
  ...livingRoomPlex,
  contact: "responding",
  sends: "ok",
  state: { level: "ok", words: "Living room Plex is up to date." },
};

function serve(nodes: NodesNode[], list: MediaserversServerView[] = [denJellyfin, livingRoomPlex]) {
  const live = liveServer();
  return {
    live,
    list,
    handlers: [...live.handlers, nodesAre(...nodes), ...mediaServerHandlers(list)],
  };
}

const servers = (node: string) =>
  page.getByRole("list", { name: `Media Servers reached from ${node}` });

test("a Node's Media Servers list the servers it reaches, each with its state", async () => {
  renderApp({
    path: `/nodes/${seedboxId}/media-servers`,
    handlers: serve([linkHome, linkSeedbox]).handlers,
  });
  await expect
    .element(
      page
        .getByRole("navigation", { name: "seedbox" })
        .getByRole("link", { name: "Media Servers" }),
    )
    .toHaveAttribute("aria-current", "page");
  await expect.element(servers("seedbox").getByText("Living room Plex")).toBeVisible();
  await expect
    .element(
      servers("seedbox").getByText(/^Living room Plex hasn't responded to Seedbox since 14:02\./u),
    )
    .toBeVisible();
  await expect.element(servers("seedbox").getByText("Den Jellyfin")).not.toBeInTheDocument();
  await expect
    .element(servers("seedbox").getByRole("link", { name: /Living room Plex/u }))
    .toHaveAttribute("href", `/settings/media-servers/${livingRoomPlex.id}`);
});

test("a Node that reaches no server says how to choose it", async () => {
  renderApp({
    path: `/nodes/${homeId}/media-servers`,
    handlers: serve([linkHome, linkSeedbox], [livingRoomPlex]).handlers,
  });
  await expect
    .element(
      page.getByText(
        "No Media Server is reached from home. Choose home under Reached from when you add or edit a server in Settings > Media Servers.",
      ),
    )
    .toBeVisible();
});

test("only a Node that stands in Media Server link has the section", async () => {
  renderApp({ path: `/nodes/${seedboxId}`, handlers: serve([linkHome, searchSeedbox]).handlers });
  const tabs = page.getByRole("navigation", { name: "seedbox" });
  await expect.element(tabs.getByRole("link", { name: "Searching" })).toBeVisible();
  await expect.element(tabs.getByRole("link", { name: "Media Servers" })).not.toBeInTheDocument();
});

test("media-servers.changed refreshes a server's state line", async () => {
  const { live, list, handlers } = serve([linkHome, linkSeedbox]);
  const app = renderApp({ path: `/nodes/${seedboxId}/media-servers`, handlers });
  await expect
    .element(servers("seedbox").getByText(/^Living room Plex hasn't responded/u))
    .toBeVisible();
  await synced(app);
  list[1] = answering;
  live.clients[0]?.send({
    event: "media-servers.changed",
    data: { serverIds: [livingRoomPlex.id] },
  });
  await expect
    .element(servers("seedbox").getByText("Living room Plex is up to date."))
    .toBeVisible();
});

test("on phone a Node's page opens its Media Servers as a pushed page", async () => {
  await asPhone();
  const { router } = renderApp({
    path: `/nodes/${seedboxId}`,
    handlers: serve([linkHome, linkSeedbox]).handlers,
  });
  await userEvent.click(page.getByRole("link", { name: "Media Servers" }));
  await expect.poll(() => router.state.location.pathname).toBe(`/nodes/${seedboxId}/media-servers`);
  await expect.element(servers("seedbox").getByText("Living room Plex")).toBeVisible();
});
