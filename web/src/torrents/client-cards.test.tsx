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
const table = () => page.getByRole("table", { name: "Torrents" });

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

test("a client that isn't responding leads the view with a red card, and its rows show when they last reported", async () => {
  open(atticAway);
  await expect
    .element(card())
    .toHaveTextContent(
      "Not respondingTransmission on homeSince 14:02Transmission isn't responding. Check that it's running and that home can reach 192.168.1.20:9091.1 torrent shows last known data.Check nowEdit client",
    );
  await expect
    .element(table().getByRole("row", { name: /^The\.Attic/u }))
    .toMatchTextContent("Last known at 14:02");
  await expect
    .element(table().getByRole("row", { name: /^The\.Attic/u }))
    .toMatchTextContent("homeTransmission");
});

test("Check now asks the Download Clients category to check every client", async () => {
  const { health } = open(atticAway);
  await userEvent.click(card().getByRole("button", { name: "Check now" }));
  await expect.element(card().getByText("Checked.")).toBeVisible();
  expect(health.checks).toEqual(["download-clients"]);
});

test("Edit client opens the client's sheet over the page", async () => {
  open(atticAway);
  await userEvent.click(card().getByRole("button", { name: "Edit client" }));
  await expect.element(page.getByRole("dialog", { name: "Edit Transmission" })).toBeVisible();
});

test("a responding client shows no card", async () => {
  open(attic);
  await expect.element(table().getByRole("row", { name: /^The\.Attic/u })).toBeVisible();
  await expect.element(card()).not.toBeInTheDocument();
});
