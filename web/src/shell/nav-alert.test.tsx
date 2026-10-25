import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { ShellAlerts, ShellAlertsNodes } from "../api/gen/model";
import { liveServer } from "../test/live-server";
import { seedboxId } from "../test/nodes";
import { renderApp } from "../test/render-app";
import { resize } from "../test/viewport";

const mainNav = () => page.getByRole("navigation", { name: "Main" });

function alerting(first: ShellAlertsNodes) {
  let nodes = first;
  return {
    handler: http.get("*/api/v1/counts", () =>
      HttpResponse.json({
        activity: 0,
        downloads: { done: 0, wanted: 0, allPaused: false },
        alerts: { nodes },
      }),
    ),
    set(next: ShellAlertsNodes) {
      nodes = next;
    },
  };
}

const dotColor = (link: Element) => {
  const dot = link.querySelector("[aria-hidden=true].rounded-capsule");
  if (dot === null) {
    throw new Error("the row draws no dot");
  }
  return getComputedStyle(dot).backgroundColor;
};

test("the Nodes row carries an amber dot while a Node is away and a red one once it lapsed, following nodes.changed", async () => {
  const server = liveServer();
  const alerts = alerting(null);
  renderApp({ path: "/library", handlers: [...server.handlers, alerts.handler] });
  const nodes = mainNav().getByRole("link", { name: /^Nodes/u });
  await expect.element(nodes).toHaveAccessibleName("Nodes");

  alerts.set("warn");
  server.clients[0]?.send({ event: "nodes.changed", data: { nodeId: seedboxId } });
  await expect.element(nodes).toHaveTextContent("Nodes needs attention");
  const amber = dotColor(nodes.element());

  alerts.set("fail");
  server.clients[0]?.send({ event: "nodes.changed", data: { nodeId: seedboxId } });
  await expect.element(nodes).toHaveTextContent("Nodes failed");
  expect(dotColor(nodes.element())).not.toBe(amber);
  expect(nodes.element().textContent).not.toMatch(/\d/u);
});

test("on the rail the Nodes icon carries the alert's dot and its name says it", async () => {
  await resize(1000, 800);
  const alerts = alerting("fail");
  renderApp({ path: "/library", handlers: [alerts.handler] });
  const nodes = mainNav().getByRole("link", { name: "Nodes, failed" });
  await expect.element(nodes).toBeVisible();
  expect(dotColor(nodes.element())).not.toBe("rgba(0, 0, 0, 0)");
});

test("the Health row sits between Nodes and Accounts, and its dot and the Tasks row's follow health.changed", async () => {
  const server = liveServer();
  let alerts: ShellAlerts = { nodes: null, health: null, tasks: null };
  renderApp({
    path: "/library",
    handlers: [
      ...server.handlers,
      http.get("*/api/v1/counts", () =>
        HttpResponse.json({
          activity: 0,
          downloads: { done: 0, wanted: 0, allPaused: false },
          alerts,
        }),
      ),
    ],
  });
  const health = mainNav().getByRole("link", { name: /^Health/u });
  const tasks = mainNav().getByRole("link", { name: /^Tasks/u });
  await expect.element(health).toHaveAccessibleName("Health");
  const system = mainNav()
    .getByRole("link")
    .all()
    .map((link) => link.element().textContent);
  expect(system.slice(system.indexOf("Nodes"), system.indexOf("Accounts") + 1)).toEqual([
    "Nodes",
    "Health",
    "Accounts",
  ]);

  alerts = { nodes: null, health: "fail", tasks: "warn" };
  server.clients[0]?.send({ event: "health.changed", data: { categories: ["scheduled-tasks"] } });
  await expect.element(health).toHaveTextContent("Health failed");
  await expect.element(tasks).toHaveTextContent("Tasks needs attention");
  expect(dotColor(health.element())).not.toBe(dotColor(tasks.element()));
});
