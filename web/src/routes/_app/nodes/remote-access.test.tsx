import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { ReachabilityCard } from "../../../api/gen/model";
import { joinCodeServer } from "../../../test/join-codes";
import { homeId } from "../../../test/nodes";
import {
  cards,
  livingRoomPlexId,
  plexSuggestion,
  remoteAccessServer,
  wanted,
} from "../../../test/remote-access";
import { renderApp } from "../../../test/render-app";
import { asPhone } from "../../../test/viewport";

function open(card: ReachabilityCard | null) {
  const server = remoteAccessServer(wanted(card));
  renderApp({
    path: "/nodes/remote-access",
    handlers: [...server.handlers, ...joinCodeServer().handlers],
  });
  return server;
}

const button = (name: string) => page.getByRole("button", { name, exact: true });

test.each([
  { card: cards.fail, chip: "Needs a relay", buttons: [] },
  { card: cards.warn, chip: "Fixable", buttons: ["Turn on Open router ports automatically"] },
])("a $chip card offers only its fix as a button", async ({ card, chip, buttons }) => {
  open(card);
  const callout = page.getByRole("status").filter({ hasText: card.subject });
  await expect.element(callout.getByText(chip, { exact: true })).toBeVisible();
  expect(
    callout
      .getByRole("button")
      .elements()
      .map((element) => element.textContent),
  ).toEqual(buttons);
});

test("the map-ports fix turns on Open router ports automatically for its Node", async () => {
  const server = open(cards.warn);
  await userEvent.click(button("Turn on Open router ports automatically"));
  await expect
    .poll(() => server.sent)
    .toContainEqual({
      what: `node ${homeId}`,
      body: { values: { "reachability.mapPorts": true } },
    });
});

test("a route nothing outside tested offers to pair a Node and raises no signal", async () => {
  open(cards.untested);
  await userEvent.click(button("Pair a Node"));
  await expect
    .element(page.getByRole("status").filter({ hasText: "Living room Plex" }))
    .not.toBeInTheDocument();
  await expect.element(page.getByRole("dialog", { name: "Add a Node" })).toBeVisible();
});

test("on phone a hop opens its checks in a sheet", async () => {
  await asPhone();
  open(cards.warn);
  await userEvent.click(button("Origin: Fails"));
  const sheet = page.getByRole("dialog", { name: "Origin · Living room Plex" });
  await expect.element(sheet.getByRole("heading", { name: "Fails over IPv4" })).toBeVisible();
  await expect.element(sheet.getByText("Forward TCP port 32400", { exact: false })).toBeVisible();
});

test("with nothing set up, choosing adds a Media Server at the address it publishes", async () => {
  const server = remoteAccessServer({
    ...wanted(null),
    nothingWanted: true,
    routes: [],
    suggestions: [plexSuggestion],
  });
  renderApp({ path: "/nodes/remote-access", handlers: server.handlers });
  await userEvent.click(button("Choose what to use away from home"));
  const suggestion = page.getByRole("listitem", { name: "Living room Plex" });
  await expect
    .element(suggestion.getByRole("textbox", { name: "Public address" }))
    .toHaveValue(plexSuggestion.public);
  await userEvent.click(suggestion.getByRole("button", { name: "Add" }));
  await expect
    .poll(() => server.sent)
    .toEqual([
      {
        what: "add",
        body: { kind: "media-server", serverId: livingRoomPlexId, public: plexSuggestion.public },
      },
    ]);
});
