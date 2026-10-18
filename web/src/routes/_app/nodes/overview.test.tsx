import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { type Locator, page, userEvent } from "vitest/browser";

import { ids, manageServer, threeNodes, vps } from "../../../test/landing";
import { nodeSystems } from "../../../test/node-system";
import { homeSystem } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";

const systems = nodeSystems({
  systems: { [ids.home]: homeSystem, [ids.vps]: homeSystem, [ids.seedbox]: "offline" },
}).handlers;
const overview = () => page.getByRole("region", { name: "Overview" });
const textOf = (locator: Locator) => () => locator.element().textContent;

function dialServer() {
  const sent: unknown[] = [];
  const handler = http.put("*/api/v1/nodes/:id/dial", async ({ request }) => {
    sent.push(await request.json());
    return HttpResponse.json(vps);
  });
  return { sent, handler };
}

test("a Worker's Overview lists its name, Roles, key, pairing and who dials, then its own report and Remove", async () => {
  renderApp({ path: `/nodes/${ids.vps}`, handlers: [...threeNodes, ...systems] });
  await expect
    .poll(textOf(overview()))
    .toMatch(
      /^OverviewNamevpsRenameRolesSearchingChange RolesKeySHA256:vps-keyPaired.+Reached byhome dials vpsChange$/u,
    );
  await expect
    .element(overview().getByRole("link", { name: "Change Roles" }))
    .toHaveAttribute("href", `/nodes/${ids.vps}/roles`);
  await expect.element(page.getByRole("region", { name: "System" })).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Remove vps" })).toBeVisible();
});

test("an offline Worker's Overview says when it was last seen and what it found, and still removes it", async () => {
  const server = manageServer({});
  const { router } = renderApp({
    path: `/nodes/${ids.seedbox}`,
    handlers: [...threeNodes, ...systems, ...server.handlers],
  });
  await expect.poll(textOf(overview())).toMatch(/Last seen.+Reached byNot connectedChange/u);
  await expect
    .element(overview().getByRole("list", { name: "What seedbox found" }))
    .toHaveTextContent("Weak: The downloads folder is on a network share.");
  await expect
    .element(
      page.getByText(
        "seedbox is offline. Its version and listeners show again when it reconnects.",
      ),
    )
    .toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Remove seedbox" }));
  const dialog = page.getByRole("dialog", { name: "Remove seedbox?" });
  await userEvent.click(dialog.getByRole("button", { name: "Remove and write off" }));
  await expect.poll(() => router.state.location.pathname).toBe("/nodes");
  expect(server.removed).toEqual([{ nodeId: ids.seedbox, writeOff: "true" }]);
});

test("Change under Reached by dials the Worker at a pasted Node address, or stops dialing it", async () => {
  const dial = dialServer();
  renderApp({ path: `/nodes/${ids.vps}`, handlers: [...threeNodes, ...systems, dial.handler] });
  await userEvent.click(overview().getByRole("button", { name: "Change" }));
  const dialog = page.getByRole("dialog", { name: "How home reaches vps" });
  await expect.element(dialog.getByText("home dials vps at 203.0.113.9:8485.")).toBeVisible();
  await userEvent.fill(dialog.getByRole("textbox", { name: "Node address" }), "SLN1-7Q2KD-M4ZP1");
  await userEvent.click(dialog.getByRole("button", { name: "home dials it" }));
  await expect.element(dialog).not.toBeInTheDocument();

  await userEvent.click(overview().getByRole("button", { name: "Change" }));
  await userEvent.click(page.getByRole("button", { name: "Stop dialing" }));
  await expect
    .element(page.getByRole("dialog", { name: "How home reaches vps" }))
    .not.toBeInTheDocument();
  expect(dial.sent).toEqual([{ nodeAddress: "SLN1-7Q2KD-M4ZP1" }, { addresses: [] }]);
});

test("the Controller's Overview has no Reached by, and its Remove row says to hand it over", async () => {
  renderApp({ path: `/nodes/${ids.home}`, handlers: [...threeNodes, ...systems] });
  await expect.element(overview().getByText("Key")).toBeVisible();
  await expect.element(overview().getByText("Reached by")).not.toBeInTheDocument();
  await expect
    .poll(textOf(page.getByRole("region", { name: "Remove" })))
    .toMatch(/home is the Controller, which keeps the library, so it can't be removed\./u);
  await expect.element(page.getByRole("button", { name: "Remove home" })).not.toBeInTheDocument();
});

test("Rename on the Overview renames the Node", async () => {
  const server = manageServer({});
  renderApp({
    path: `/nodes/${ids.vps}`,
    handlers: [...threeNodes, ...systems, ...server.handlers],
  });
  await userEvent.click(overview().getByRole("button", { name: "Rename" }));
  const dialog = page.getByRole("dialog", { name: "Rename vps" });
  await userEvent.fill(dialog.getByRole("textbox", { name: "Name" }), "edge");
  await userEvent.click(dialog.getByRole("button", { name: "Rename" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(server.renamed).toEqual([{ nodeId: ids.vps, body: { name: "edge" } }]);
});
