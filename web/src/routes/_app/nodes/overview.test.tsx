import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { type Locator, page, userEvent } from "vitest/browser";

import { getGetSupersededMockHandler } from "../../../api/gen/query/superseded/superseded.msw";
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

test("a Worker's Overview lists its name, key, pairing and who dials, then its own report and Remove", async () => {
  renderApp({ path: `/nodes/${ids.vps}`, handlers: [...threeNodes, ...systems] });
  await expect
    .poll(textOf(overview()))
    .toMatch(
      /^OverviewNamevpsRenameKeySHA256:vps-keyPaired.+Connectionhome connects to vpsChange$/u,
    );
  await expect.element(page.getByRole("region", { name: "System" })).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Remove vps" })).toBeVisible();
});

test("an offline Worker's Overview says what it found, and still removes it", async () => {
  const server = manageServer({});
  const { router } = renderApp({
    path: `/nodes/${ids.seedbox}`,
    handlers: [...threeNodes, ...systems, ...server.handlers],
  });
  await expect.poll(textOf(overview())).toMatch(/ConnectionNot connectedChange/u);
  await expect
    .element(overview().getByRole("list", { name: "What seedbox found" }))
    .toMatchTextContent(/^Weak.+network share/u);
  await expect
    .element(
      page.getByText(
        "seedbox is offline. Its version and listeners show again when it reconnects.",
      ),
    )
    .toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Remove seedbox" }));
  const dialog = page.getByRole("dialog", { name: "Remove seedbox?" });
  await userEvent.click(dialog.getByRole("button", { name: "Remove and give up its work" }));
  await expect.poll(() => router.state.location.pathname).toBe("/nodes");
  expect(server.removed).toEqual([{ nodeId: ids.seedbox, writeOff: "true" }]);
});

test("Change under Connection dials the Worker at a pasted Node address, or stops dialing it", async () => {
  const dial = dialServer();
  renderApp({ path: `/nodes/${ids.vps}`, handlers: [...threeNodes, ...systems, dial.handler] });
  await userEvent.click(overview().getByRole("button", { name: "Change" }));
  const dialog = page.getByRole("dialog", { name: "How home connects to vps" });
  await expect.element(dialog.getByText("home connects to vps at 203.0.113.9:8485.")).toBeVisible();
  await userEvent.fill(dialog.getByRole("textbox", { name: "Node address" }), "SLN1-7Q2KD-M4ZP1");
  await userEvent.click(dialog.getByRole("button", { name: "Connect from home" }));
  await expect.element(dialog).not.toBeInTheDocument();

  await userEvent.click(overview().getByRole("button", { name: "Change" }));
  await userEvent.click(page.getByRole("button", { name: "Stop connecting from home" }));
  await expect
    .element(page.getByRole("dialog", { name: "How home connects to vps" }))
    .not.toBeInTheDocument();
  expect(dial.sent).toEqual([{ nodeAddress: "SLN1-7Q2KD-M4ZP1" }, { addresses: [] }]);
});

test("the Controller's Overview has no Connection, and its Remove row says why it stays", async () => {
  renderApp({ path: `/nodes/${ids.home}`, handlers: [...threeNodes, ...systems] });
  await expect.element(overview().getByText("Key")).toBeVisible();
  await expect.element(overview().getByText("Connection")).not.toBeInTheDocument();
  await expect
    .poll(textOf(page.getByRole("region", { name: "Remove" })))
    .toMatch(/home is the Controller and keeps the library, so it can't be removed\./u);
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

test("the Overview of the Worker that said another Controller replaced this one offers Stop, and stops once confirmed", async () => {
  let stopped = 0;
  renderApp({
    path: `/nodes/${ids.vps}`,
    handlers: [
      ...threeNodes,
      ...systems,
      getGetSupersededMockHandler({
        state: "unproven",
        at: new Date().toISOString(),
        replacement: { key: "EEGV-S8BN-TGNW-PSQ0", address: "192.168.1.20" },
        worker: "vps",
        workerId: ids.vps,
      }),
      http.post("*/api/v1/superseded/stop", () => {
        stopped++;
        return new HttpResponse(null, { status: 204 });
      }),
    ],
  });
  const section = page.getByRole("region", { name: "Another Controller" });
  await expect
    .element(
      section.getByText(
        "vps now connects to a different Controller and sent no proof of a restore. If you restored a Backup on another machine, stop this Controller.",
      ),
    )
    .toBeVisible();
  await userEvent.click(section.getByRole("button", { name: "Stop this Controller…" }));
  const dialog = page.getByRole("dialog", { name: "Stop this Controller?" });
  await expect
    .element(
      dialog.getByText(
        "This Controller pauses every torrent, stops searching and shows only one page, where you can wipe it and join the new Controller. To undo this, select Resume this Controller on that page.",
      ),
    )
    .toBeVisible();
  expect(stopped).toBe(0);
  await userEvent.click(dialog.getByRole("button", { name: "Stop", exact: true }));
  await expect.poll(() => stopped).toBe(1);
});

test("another Node's Overview offers no Stop", async () => {
  renderApp({
    path: `/nodes/${ids.seedbox}`,
    handlers: [
      ...threeNodes,
      ...systems,
      getGetSupersededMockHandler({ state: "unproven", worker: "vps", workerId: ids.vps }),
    ],
  });
  await expect.element(page.getByRole("button", { name: "Remove seedbox" })).toBeVisible();
  await expect
    .element(page.getByRole("region", { name: "Another Controller" }))
    .not.toBeInTheDocument();
});
