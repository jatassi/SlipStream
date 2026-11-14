import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { mediaServersWorld } from "../test/media-server-writes";
import { denJellyfin, livingRoomPlex } from "../test/media-servers";
import { home, nodesAre } from "../test/nodes";
import { renderApp } from "../test/render-app";

const path = "/settings/media-servers";

test("the section shows the Plex account, and signing out removes every Plex server", async () => {
  const world = mediaServersWorld([denJellyfin, livingRoomPlex]);
  renderApp({ path, handlers: [nodesAre(home), ...world.handlers] });
  const account = page.getByRole("region", { name: "Plex account" });
  await expect.element(account.getByText("ada")).toBeVisible();
  await expect.element(account.getByText("Plex Pass")).toBeVisible();
  const servers = page.getByRole("list", { name: "Media Servers" });
  await expect.element(servers.getByText("Living room Plex")).toBeVisible();

  await userEvent.click(account.getByRole("button", { name: "Sign out" }));
  const dialog = page.getByRole("dialog", { name: "Sign out of Plex?" });
  await expect
    .element(
      dialog.getByText(
        "Signing out also removes Living room Plex from SlipStream, with its waiting changes and play history. Nothing changes on the servers themselves.",
      ),
    )
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Sign out" }));
  await expect.element(page.getByText("Signed out of Plex")).toBeVisible();
  await expect.element(servers.getByText("Living room Plex")).not.toBeInTheDocument();
  await expect.element(servers.getByText("Den Jellyfin")).toBeVisible();
  await expect.element(account).not.toBeInTheDocument();
  expect(world.writes).toEqual([{ path: "plex-sign-out", body: null }]);
});

test("the section shows no Plex account while SlipStream is signed in to none", async () => {
  const world = mediaServersWorld([denJellyfin], null);
  renderApp({ path, handlers: [nodesAre(home), ...world.handlers] });
  await expect.element(page.getByText("Den Jellyfin")).toBeVisible();
  expect(page.getByRole("region", { name: "Plex account" }).query()).toBeNull();
});
