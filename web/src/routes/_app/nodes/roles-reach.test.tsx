import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { NodesNode } from "../../../api/gen/model";
import { home, homeId, nodesAre, seedbox, seedboxId } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";
import { board, boardIs, homeSeat, outcome, rolesServer, seedboxSeat } from "../../../test/roles";

const seedboxNamed = { id: seedboxId, name: "seedbox" };

const reachedTwice = outcome({
  effects: [
    {
      kind: "gains",
      node: seedboxNamed,
      role: "notifying",
      words: "seedbox gets the Notifying Role.",
    },
    {
      kind: "lets-go",
      node: { id: homeId, name: "home" },
      role: "notifying",
      words: "home stops Notifying.",
    },
  ],
  checks: [
    {
      node: seedboxNamed,
      role: "notifying",
      target: "smtp.example.test:587",
      verdict: "allow",
      reason: "reached",
      words: "seedbox reached smtp.example.test:587 in 12 ms.",
    },
    {
      node: seedboxNamed,
      role: "notifying",
      target: "gotify.lan:443",
      verdict: "warn",
      reason: "reach_warned",
      words:
        "seedbox can't reach gotify.lan. Messages to Gotify 'Home' fail while seedbox sends them.",
    },
    {
      node: seedboxNamed,
      role: "notifying",
      target: "ntfy.lan:443",
      verdict: "warn",
      reason: "reach_warned",
      words:
        "seedbox can't reach ntfy.lan. Messages to ntfy 'Phone' fail while seedbox sends them.",
    },
  ],
});

test("a connection host the Node can't reach warns, and Apply stays on", async () => {
  const onlineSeedbox: NodesNode = {
    ...seedbox,
    standing: "in-step",
    link: { state: "up", since: "2026-10-12T10:00:00Z" },
  };
  const roles = rolesServer({ plans: [reachedTwice], sets: [] });
  renderApp({
    path: `/nodes/${seedboxSeat.id}/roles`,
    handlers: [
      nodesAre(home, onlineSeedbox),
      boardIs(board(homeSeat, seedboxSeat)),
      ...roles.handlers,
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Connect", exact: true }));
  await userEvent.click(page.getByRole("checkbox", { name: "Notifying", exact: true }));
  await userEvent.click(page.getByRole("button", { name: "Review" }));
  const dialog = page.getByRole("dialog", { name: "Review the change" });
  const checks = dialog.getByRole("list", { name: "Checks" });
  await expect.element(checks.getByText(/^seedbox reached smtp/u)).toBeVisible();
  await expect.element(checks.getByText(/Gotify 'Home'/u)).toBeVisible();
  await expect.element(checks.getByText(/ntfy 'Phone'/u)).toBeVisible();
  expect(checks.getByText("Warning").elements()).toHaveLength(2);
  await expect.element(dialog.getByRole("button", { name: "Apply" })).toBeEnabled();
});
