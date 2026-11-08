import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { downloadClients, homeCard } from "../test/download-clients";
import { attic, withExternals } from "../test/external-clients";
import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { torrent, torrentsServer } from "../test/torrents";

const table = () => page.getByRole("table", { name: "Torrents" });

const builtIn = torrent({
  state: "downloading",
  tabs: ["downloading", "all"],
  name: "The.Quiet.Orchard.2021.1080p.WEB-DL",
});

test("the client filter narrows the table to one client and writes the address", async () => {
  const inTransmission = torrent({
    state: "downloading",
    tabs: ["downloading", "all"],
    name: "The.Attic.2024.1080p.WEB-DL",
    clientId: attic.id,
    clientName: "Transmission",
    clientKind: "transmission",
  });
  const live = liveServer();
  const server = torrentsServer(
    [builtIn, inTransmission],
    undefined,
    downloadClients({ cards: [withExternals(homeCard, attic)] }),
  );
  const app = renderApp({
    path: "/activity/torrents",
    handlers: [...live.handlers, ...server.handlers],
  });
  await expect.poll(() => table().getByRole("row").all()).toHaveLength(3);

  await userEvent.click(page.getByRole("combobox", { name: "Client" }));
  await userEvent.click(page.getByRole("option", { name: "Transmission on home" }));
  await expect.poll(() => table().getByRole("row").all()).toHaveLength(2);
  await expect.element(table().getByRole("row", { name: /^The\.Attic/u })).toBeVisible();
  expect(app.router.state.location.search).toEqual({ client: attic.id });

  await userEvent.click(page.getByRole("combobox", { name: "Client" }));
  await userEvent.click(page.getByRole("option", { name: "Built-in client" }));
  await expect.element(table().getByRole("row", { name: /^The\.Quiet\.Orchard/u })).toBeVisible();
  expect(table().getByRole("row").all()).toHaveLength(2);
  expect(app.router.state.location.search).toEqual({ client: "built-in" });
});
