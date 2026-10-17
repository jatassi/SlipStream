import { HttpResponse } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { liveServer } from "../../../test/live-server";
import { homeConfig, nodeSystems, withPort } from "../../../test/node-system";
import { home, homeId, homeSystem, nodesAre, seedbox, seedboxId } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";
import { worker } from "../../../test/worker";
import { asPhone } from "../../../test/viewport";

const field = (name: string) => page.getByRole("textbox", { name });

test("Web and API shows the port with its default and Ports to try", async () => {
  renderApp({ path: `/nodes/${homeId}/web`, handlers: nodeSystems().handlers });
  await expect.element(field("Port")).toHaveValue("8484");
  await expect.element(page.getByText("Default 8484.")).toBeVisible();
  await expect.element(field("Ports to try")).toHaveValue("10");
  await expect.element(page.getByText("Default 10 ports. Between 1 and 100 ports.")).toBeVisible();
});

test("a port set by the environment is locked and names its variable", async () => {
  const locked = withPort({ source: "env", lockedBy: "SLIPSTREAM_HTTP_LISTEN", value: 9000 });
  renderApp({
    path: `/nodes/${homeId}/web`,
    handlers: nodeSystems({ configs: { [homeId]: locked } }).handlers,
  });
  await expect.element(page.getByText("Set by SLIPSTREAM_HTTP_LISTEN")).toBeVisible();
  await expect.element(field("Port")).toBeDisabled();
});

test("saving a port toasts, and once saved it asks for a restart to use it", async () => {
  const server = liveServer();
  const systems = nodeSystems();
  const app = renderApp({
    path: `/nodes/${homeId}/web`,
    handlers: [...server.handlers, ...systems.handlers],
  });
  await expect.poll(() => app.live.getState()).toBe("open");
  await userEvent.fill(field("Port"), "9000");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Web and API settings saved")).toBeVisible();
  expect(systems.writes).toEqual([
    { method: "PATCH", nodeId: homeId, body: { "http.listen": 9000 } },
  ]);

  systems.state.configs[homeId] = withPort({ value: 9000, isDefault: false, pendingRestart: true });
  server.clients[0]?.send({ event: "node.settings-changed", data: { nodeId: homeId } });
  await expect.element(page.getByText("Restart to use port 9000")).toBeVisible();
  await expect
    .element(page.getByText("SlipStream listens on 8484 until it restarts."))
    .toBeVisible();
  await expect.element(page.getByRole("button", { name: "Restart now" })).toBeVisible();
  await expect.element(field("Port")).toHaveValue("9000");
});

test("a saved port shows from the save's answer, without waiting for the live event", async () => {
  const systems = nodeSystems();
  renderApp({ path: `/nodes/${homeId}/web`, handlers: systems.handlers });
  await expect.element(field("Port")).toHaveValue("8484");
  systems.state.configs[homeId] = withPort({ value: 9000, isDefault: false, pendingRestart: true });
  await userEvent.fill(field("Port"), "9000");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Restart to use port 9000")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Save" })).not.toBeInTheDocument();
});

test("on Docker the pending port says the container's mapping must change too", async () => {
  const docker = {
    ...home,
    link: {
      ...home.link,
      build: { version: "0.4.0", os: "linux", arch: "amd64", installKind: "docker" },
    },
  };
  const pending = withPort({ value: 9000, isDefault: false, pendingRestart: true });
  renderApp({
    path: `/nodes/${homeId}/web`,
    handlers: [nodesAre(docker), ...nodeSystems({ configs: { [homeId]: pending } }).handlers],
  });
  await expect
    .element(page.getByText("Change the container's -p mapping to publish 9000 too, then restart."))
    .toBeVisible();
});

test("a port the Node refuses shows the problem's sentence under the field", async () => {
  const systems = nodeSystems();
  systems.state.refuse = HttpResponse.json(
    {
      status: 422,
      code: "port_used_by_node",
      title: "The Node uses this port already",
      detail: "The Node link uses 8485.",
      errors: [{ location: "body.http.listen", message: "The Node link uses 8485." }],
    },
    { status: 422, headers: { "Content-Type": "application/problem+json" } },
  );
  renderApp({ path: `/nodes/${homeId}/web`, handlers: systems.handlers });
  await userEvent.fill(field("Port"), "8485");
  await userEvent.click(page.getByRole("button", { name: "Save" }));
  await expect.element(page.getByRole("alert")).toHaveTextContent("The Node link uses 8485.");
});

test("Log settings saves the level at once and shows the log folder and Download logs", async () => {
  const systems = nodeSystems();
  renderApp({ path: `/nodes/${homeId}/log-settings`, handlers: systems.handlers });
  await expect.element(field("Log folder")).toHaveValue("/data/home/logs");
  await expect.element(page.getByRole("button", { name: "Download logs" })).toBeVisible();
  await expect.element(field("Days kept")).toHaveValue("30");
  await expect.element(page.getByRole("switch", { name: "Compress rotated files" })).toBeChecked();

  await userEvent.click(page.getByRole("combobox", { name: "Level" }));
  await userEvent.click(page.getByRole("option", { name: "Debug" }));
  await expect.element(page.getByText("Log settings saved")).toBeVisible();
  expect(systems.writes).toEqual([
    { method: "PATCH", nodeId: homeId, body: { "log.level": "debug" } },
  ]);
});

test("after a fallback the Overview names both ports", async () => {
  const fellBack = {
    ...homeSystem,
    http: { configured: "[::]:8484" },
    listeners: { ...homeSystem.listeners, http: "[::]:8486" },
  };
  renderApp({
    path: `/nodes/${homeId}`,
    handlers: nodeSystems({ systems: { [homeId]: fellBack } }).handlers,
  });
  await expect
    .element(page.getByText("Port 8484 was taken, so SlipStream is on 8486."))
    .toBeVisible();
});

test("a Worker's sections read its own facts and settings", async () => {
  const seedboxSystem = { ...homeSystem, dataDir: "/srv/seedbox", logsDir: "/srv/seedbox/logs" };
  renderApp({
    path: `/nodes/${seedboxId}/log-settings`,
    handlers: [
      nodesAre(home, { ...seedbox, link: { state: "up", since: "2026-10-12T10:00:00Z" } }),
      ...nodeSystems({
        systems: { [homeId]: homeSystem, [seedboxId]: seedboxSystem },
        configs: { [homeId]: homeConfig, [seedboxId]: homeConfig },
      }).handlers,
    ],
  });
  await expect.element(field("Log folder")).toHaveValue("/srv/seedbox/logs");
});

test("an offline Worker's settings say it is offline, on phone too", async () => {
  await asPhone();
  renderApp({
    path: `/nodes/${seedboxId}/web`,
    handlers: [
      nodesAre(home, seedbox),
      ...nodeSystems({ configs: { [homeId]: homeConfig, [seedboxId]: "offline" } }).handlers,
    ],
  });
  await expect
    .element(
      page.getByText("seedbox is offline. Its settings and logs come back when it reconnects."),
    )
    .toBeVisible();
});

test("an offline Worker's section loads once its link is back", async () => {
  const server = liveServer();
  const systems = nodeSystems({ configs: { [homeId]: homeConfig, [seedboxId]: "offline" } });
  const app = renderApp({
    path: `/nodes/${seedboxId}/web`,
    handlers: [...server.handlers, nodesAre(home, seedbox), ...systems.handlers],
  });
  await expect.poll(() => app.live.getState()).toBe("open");
  await expect.element(page.getByText(/^seedbox is offline\./u)).toBeVisible();

  systems.state.configs[seedboxId] = homeConfig;
  worker.use(nodesAre(home, { ...seedbox, link: { state: "up", since: "2026-10-12T11:00:00Z" } }));
  server.clients[0]?.send({ event: "nodes.changed", data: { nodeId: seedboxId } });
  await expect.element(field("Port")).toHaveValue("8484");
});
