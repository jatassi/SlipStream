import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { ReachabilityCard } from "../../../api/gen/model";
import { joinCodeServer } from "../../../test/join-codes";
import { homeId } from "../../../test/nodes";
import { cards, remoteAccessServer, wanted } from "../../../test/remote-access";
import { renderApp } from "../../../test/render-app";

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
