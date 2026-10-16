import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { liveServer } from "../../test/live-server";
import { nodeSystems, restarts } from "../../test/node-system";
import { home, homeId, homeSystem, nodesAre, seedbox, seedboxId } from "../../test/nodes";
import { renderApp } from "../../test/render-app";
import { asPhone } from "../../test/viewport";
import { worker } from "../../test/worker";
import { paletteShortcut } from "../palette-state";

const said =
  "The web app reconnects by itself. Downloads and imports pause and pick up where they left off.";
const confirm = (name: string) => page.getByRole("dialog", { name });

test("the account menu asks before restarting SlipStream, and Cancel restarts nothing", async () => {
  const restart = restarts();
  renderApp({ path: "/library", handlers: [...nodeSystems().handlers, restart.handler] });
  await userEvent.click(page.getByRole("button", { name: "Account" }));
  await expect
    .element(page.getByRole("menu"))
    .toMatchTextContent(/Startup sequence.*Restart SlipStream….*Sign out$/u);
  await userEvent.click(page.getByRole("menuitem", { name: "Restart SlipStream…" }));

  await expect.element(confirm("Restart SlipStream?")).toMatchTextContent(said);
  await expect
    .element(confirm("Restart SlipStream?").getByRole("button", { name: "Restart" }))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Cancel" }));
  await expect.element(confirm("Restart SlipStream?")).not.toBeInTheDocument();
  expect(restart.asked).toEqual([]);
});

test("the palette runs Restart SlipStream…", async () => {
  renderApp({ path: "/library", handlers: nodeSystems().handlers });
  const [modifier] = paletteShortcut.keys.split("+");
  await expect.element(page.getByRole("banner")).toBeVisible();
  await userEvent.keyboard(`{${modifier}>}k{/${modifier}}`);
  await userEvent.keyboard("restart");
  await userEvent.click(page.getByRole("option", { name: "Restart SlipStream…" }));
  await expect.element(confirm("Restart SlipStream?")).toMatchTextContent(said);
  await userEvent.click(page.getByRole("button", { name: "Cancel" }));
});

test("More on a phone offers Restart SlipStream…", async () => {
  await asPhone();
  renderApp({ path: "/library", handlers: nodeSystems().handlers });
  await userEvent.click(page.getByRole("button", { name: "More" }));
  await userEvent.click(
    page.getByRole("dialog", { name: "More" }).getByRole("button", { name: "Restart SlipStream…" }),
  );
  await expect.element(confirm("Restart SlipStream?")).toMatchTextContent(said);
  await userEvent.click(page.getByRole("button", { name: "Cancel" }));
});

test("a Worker restarted from Nodes shows Restarting until its link is up again", async () => {
  const server = liveServer();
  const restart = restarts();
  const seedboxUp = { ...seedbox, link: { state: "up", since: "2026-10-12T10:00:00Z" } } as const;
  const app = renderApp({
    path: `/nodes/${seedboxId}`,
    handlers: [
      ...server.handlers,
      nodesAre(home, seedboxUp),
      ...nodeSystems({ systems: { [homeId]: homeSystem, [seedboxId]: homeSystem } }).handlers,
      restart.handler,
    ],
  });
  await expect.poll(() => app.live.getState()).toBe("open");
  await userEvent.click(page.getByRole("button", { name: "Restart", exact: true }));
  await expect.element(confirm("Restart seedbox?")).toMatchTextContent(said);
  await userEvent.click(confirm("Restart seedbox?").getByRole("button", { name: "Restart" }));

  await expect.element(page.getByText("seedbox is restarting…")).toBeVisible();
  expect(restart.asked).toEqual([seedboxId]);
  await expect
    .element(page.getByRole("combobox", { name: "Node" }))
    .toHaveTextContent("restartingseedbox");
  await expect.element(page.getByRole("button", { name: "Restart", exact: true })).toBeDisabled();

  const back = new Date(Date.now() + 1000).toISOString();
  worker.use(nodesAre(home, { ...seedbox, link: { state: "up", since: back } }));
  server.clients[0]?.send({ event: "nodes.changed", data: { nodeId: seedboxId } });
  await expect.element(page.getByText("127.0.0.2:8080")).toBeVisible();
  await expect.element(page.getByText("seedbox is restarting…")).not.toBeInTheDocument();
});
