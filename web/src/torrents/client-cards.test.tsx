import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { downloadClients, homeCard } from "../test/download-clients";
import { attic, atticAway, withExternals } from "../test/external-clients";
import { healthServer, reportOf } from "../test/health";
import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { torrent, torrentsServer } from "../test/torrents";

const stale = torrent({
  state: "downloading",
  tabs: ["downloading", "all"],
  name: "The.Attic.2024.1080p.WEB-DL",
  clientId: attic.id,
  clientName: "Transmission",
  clientKind: "transmission",
  lastKnown: "2026-11-07T14:02:00Z",
  lastKnownCause: "client",
});
const builtIn = torrent({ state: "downloading", tabs: ["downloading", "all"] });

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"], now: new Date("2026-11-07T15:00:00Z") });
});

afterEach(() => {
  vi.useRealTimers();
});

const card = () => page.getByRole("region", { name: "Transmission on home" });

function open(client: typeof attic) {
  const health = healthServer(reportOf());
  const clients = downloadClients({ cards: [withExternals(homeCard, client)] });
  renderApp({
    path: "/activity/torrents",
    handlers: [
      ...health.handlers,
      ...liveServer().handlers,
      ...torrentsServer([stale, builtIn], undefined, clients).handlers,
    ],
  });
  return { health };
}

test("Edit client opens the client's sheet over the page", async () => {
  open(atticAway);
  await userEvent.click(card().getByRole("button", { name: "Edit client" }));
  await expect.element(page.getByRole("dialog", { name: "Edit Transmission" })).toBeVisible();
});
