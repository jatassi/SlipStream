import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { WorksMonitoredChange } from "../api/gen/model";
import { getSetWorksMonitoredMockHandler } from "../api/gen/query/works/works.msw";
import { SetWorksMonitoredBody } from "../api/gen/zod/works/works.zod";
import { orchardRow } from "../test/library";
import { longShelfRow, mixedLibrary, monitoringServer, shortTalesRow } from "../test/monitoring";
import { renderApp } from "../test/render-app";

const bar = page.getByRole("toolbar", { name: "Picked Works" });

function bulkServer() {
  const monitoring = monitoringServer(({ preset }) => ({
    start: 0,
    stop: preset === "future" ? 20 : 0,
  }));
  const monitored: WorksMonitoredChange[] = [];
  const handlers = [
    ...mixedLibrary,
    monitoring.handler,
    getSetWorksMonitoredMockHandler(async ({ request }) => {
      const body = SetWorksMonitoredBody.parse(await request.json());
      monitored.push(body);
      return { outcomes: body.workIds.map((workId) => ({ workId, outcome: "done" as const })) };
    }),
  ];
  return { monitoring, monitored, handlers };
}

async function startPicking() {
  await userEvent.click(page.getByRole("button", { name: "Library options" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Select" }));
}

const pickBox = (label: string) => page.getByRole("checkbox", { name: `Pick ${label}` });

test("Set monitoring previews one preset across the picked Works and applies it to each", async () => {
  const server = bulkServer();
  renderApp({ path: "/library", handlers: server.handlers });
  await startPicking();
  await userEvent.click(pickBox(longShelfRow.label));
  await userEvent.click(pickBox(shortTalesRow.label));
  await expect.element(bar.getByText("2 Anthologies picked")).toBeVisible();

  await userEvent.click(bar.getByRole("button", { name: "Set monitoring…" }));
  const dialog = page.getByRole("dialog", {
    name: "Set monitoring for The Long Shelf and Short Tales",
  });
  const choices = dialog.getByRole("radiogroup", { name: "Stories to monitor" });
  await expect.element(choices.getByRole("radio", { name: "All · 2352 stories" })).toBeChecked();
  await expect.element(dialog.getByRole("switch", { name: "Include extras" })).not.toBeChecked();

  await userEvent.click(choices.getByRole("radio", { name: "Future · 0 stories" }));
  await expect
    .element(dialog.getByText("40 stories stop being monitored across 2 anthologies."))
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Apply" }));
  await expect
    .element(page.getByText("Changed the monitoring of The Long Shelf and Short Tales."))
    .toBeVisible();
  expect(server.monitoring.sent.at(-1)).toEqual({
    workIds: [longShelfRow.id, shortTalesRow.id],
    preset: "future",
    includeAsides: false,
    preview: false,
  });
});

test("Set monitoring is offered only while every pick has many Items", async () => {
  renderApp({ path: "/library", handlers: bulkServer().handlers });
  await startPicking();
  await userEvent.click(pickBox(orchardRow.label));
  await expect.element(bar.getByRole("button", { name: "Monitor", exact: true })).toBeVisible();
  await expect
    .element(bar.getByRole("button", { name: "Set monitoring…" }))
    .not.toBeInTheDocument();

  await userEvent.click(pickBox(longShelfRow.label));
  await expect.element(bar.getByText("2 Works picked")).toBeVisible();
  await expect
    .element(bar.getByRole("button", { name: "Set monitoring…" }))
    .not.toBeInTheDocument();
});

test("bulk Monitor and Unmonitor still send on and off, which mean All and None", async () => {
  const server = bulkServer();
  renderApp({ path: "/library", handlers: server.handlers });
  await startPicking();
  await userEvent.click(pickBox(longShelfRow.label));
  await userEvent.click(bar.getByRole("button", { name: "Monitor", exact: true }));
  await expect.element(page.getByText("Monitored The Long Shelf.")).toBeVisible();
  await userEvent.click(bar.getByRole("button", { name: "Unmonitor" }));
  await expect.element(page.getByText("Stopped monitoring The Long Shelf.")).toBeVisible();
  expect(server.monitored).toEqual([
    { workIds: [longShelfRow.id], on: true },
    { workIds: [longShelfRow.id], on: false },
  ]);
});
