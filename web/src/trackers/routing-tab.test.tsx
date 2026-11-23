import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { homeCard } from "../test/download-clients";
import { attic, withExternals } from "../test/external-clients";
import { homeId, seedboxId } from "../test/nodes";
import { renderApp } from "../test/render-app";
import { homeRoute, routingServer } from "../test/tracker-routes";
import { seedpoolId } from "../test/trackers";

const routingPage = `/settings/trackers/${seedpoolId}/routing`;
const homeRow = () => page.getByRole("region", { name: "Indexers on home" });
const picker = () => homeRow().getByRole("combobox", { name: "Grabs go to" });

test("a refused pick shows its reason on the row and keeps the route", async () => {
  const detail = "No Indexer on home reaches seedpool. Pick the home Node of one of its Indexers.";
  const server = routingServer({
    refuse: {
      status: 409,
      code: "route_not_a_home",
      title: "No Indexer on that Node reaches this Tracker",
      detail,
    },
  });
  renderApp({ path: routingPage, handlers: server.handlers });
  await userEvent.click(picker());
  await userEvent.click(page.getByRole("option", { name: "Built-in client on seedbox" }));
  await expect.element(homeRow().getByRole("alert")).toHaveTextContent(detail);
  await expect
    .element(homeRow().getByText("These Grabs go to the built-in client on home, the default."))
    .toBeVisible();
  expect(server.writes).toEqual([
    { method: "PUT", nodeId: homeId, body: { builtInOn: seedboxId } },
  ]);
});

test("a refusal with no detail, such as node_not_downloading, shows its title on the row", async () => {
  const server = routingServer({
    refuse: {
      status: 409,
      code: "node_not_downloading",
      title: "That Node doesn't have the Downloading Role",
    },
  });
  renderApp({ path: routingPage, handlers: server.handlers });
  await userEvent.click(picker());
  await userEvent.click(page.getByRole("option", { name: "Built-in client on seedbox" }));
  await expect
    .element(homeRow().getByRole("alert"))
    .toMatchTextContent(/doesn't have the Downloading Role/u);
});

test("a skipped pick says why its Grabs go to the default", async () => {
  const server = routingServer({
    routes: [{ ...homeRoute, stored: { builtInOn: seedboxId }, skipped: true }],
  });
  renderApp({ path: routingPage, handlers: server.handlers });
  await expect
    .element(
      homeRow().getByText(
        "seedbox no longer has the Downloading Role, so these Grabs go to the built-in client on home, the default.",
      ),
    )
    .toBeVisible();
  await expect.element(homeRow().getByRole("button", { name: "Use the default" })).toBeVisible();
});

test("a pick of an external client that is off says so", async () => {
  const off = { ...attic, on: false };
  const server = routingServer({
    routes: [{ ...homeRoute, stored: { clientId: off.id }, skipped: true }],
    cards: [withExternals(homeCard, off)],
  });
  renderApp({ path: routingPage, handlers: server.handlers });
  await expect
    .element(
      homeRow().getByText(
        "Transmission is off, so these Grabs go to the built-in client on home, the default.",
      ),
    )
    .toBeVisible();
});

test("a home without the Downloading Role sends its Grabs to the client set for each Work's library root", async () => {
  const server = routingServer({ routes: [{ ...homeRoute, to: null, default: null }] });
  renderApp({ path: routingPage, handlers: server.handlers });
  await expect
    .element(
      homeRow().getByText(
        "home doesn't have the Downloading Role, so these Grabs go to the client set for each Work's library root.",
      ),
    )
    .toBeVisible();
  await expect.element(picker()).toHaveTextContent("Client set for each Work's library root");
});

test("a public Tracker's Routing tab says its Grabs follow the library root", async () => {
  const server = routingServer();
  server.trackers.trackers = server.trackers.trackers.map((tracker) =>
    tracker.id === seedpoolId ? { ...tracker, privacy: "public" } : tracker,
  );
  renderApp({ path: routingPage, handlers: server.handlers });
  await expect
    .element(
      page.getByText(
        "seedpool is public. Its Grabs go to the client set for each Work's library root.",
      ),
    )
    .toBeVisible();
  await expect.element(page.getByRole("combobox")).not.toBeInTheDocument();
});
