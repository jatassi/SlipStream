import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { WorksMonitoredChange } from "../../../api/gen/model";
import { getSetWorksMonitoredMockHandler } from "../../../api/gen/query/works/works.msw";
import { SetWorksMonitoredBody } from "../../../api/gen/zod/works/works.zod";
import { orchardRow } from "../../../test/library";
import { cinemaLibrary, ember, harvest } from "../../../test/library-page";
import { renderApp } from "../../../test/render-app";

const bar = page.getByRole("toolbar", { name: "Picked Works" });

function monitorServer() {
  const sent: WorksMonitoredChange[] = [];
  const handler = getSetWorksMonitoredMockHandler(async ({ request }) => {
    const body = SetWorksMonitoredBody.parse(await request.json());
    sent.push(body);
    return { outcomes: body.workIds.map((workId) => ({ workId, outcome: "done" as const })) };
  });
  return { sent, handler };
}

test("while picking a click picks, and Monitor acts on every pick, the ones filters hide too", async () => {
  const monitor = monitorServer();
  const { router } = renderApp({
    path: "/library",
    handlers: [monitor.handler, ...cinemaLibrary().handlers],
  });
  await userEvent.click(page.getByRole("button", { name: "Library options" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Select" }));
  await expect.element(bar.getByText("Pick Works to change them together.")).toBeVisible();

  await userEvent.click(page.getByRole("link", { name: ember.label }));
  await userEvent.click(page.getByRole("checkbox", { name: `Pick ${orchardRow.label}` }));
  await expect.element(bar.getByText("2 Films picked")).toBeVisible();
  expect(router.state.location.pathname).toBe("/library");

  await userEvent.click(page.getByRole("button", { name: "Filter" }));
  await userEvent.click(page.getByRole("checkbox", { name: "Not monitored (1)" }));
  await userEvent.keyboard("{Escape}");
  await expect.element(bar.getByText("· 1 hidden by filters")).toBeVisible();

  await userEvent.click(bar.getByRole("button", { name: "Unmonitor" }));
  await expect
    .element(page.getByText(`Stopped monitoring ${ember.label} and ${orchardRow.label}.`))
    .toBeVisible();
  expect(monitor.sent).toEqual([{ workIds: [ember.id, orchardRow.id], on: false }]);
});

test("Select all picks every Work shown, and Done leaves picking with nothing picked", async () => {
  const { router } = renderApp({ path: "/library", handlers: cinemaLibrary().handlers });
  await userEvent.click(page.getByRole("checkbox", { name: `Pick ${harvest.label}` }));
  await expect.element(bar.getByText("1 Film picked")).toBeVisible();
  await userEvent.click(bar.getByRole("button", { name: "Select all" }));
  await expect.element(bar.getByText("3 Films picked")).toBeVisible();
  await userEvent.click(bar.getByRole("button", { name: "Select all" }));
  await expect.element(bar.getByText("Pick Works to change them together.")).toBeVisible();

  await userEvent.click(bar.getByRole("button", { name: "Done" }));
  await expect.element(bar).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("link", { name: harvest.label }));
  await expect.poll(() => router.state.location.pathname).toBe(`/works/${harvest.id}`);
});

test("a Shift-click picks every Work between it and the last pick", async () => {
  renderApp({ path: "/library?view=table", handlers: cinemaLibrary().handlers });
  await userEvent.click(page.getByRole("checkbox", { name: `Pick ${ember.label}` }));
  await userEvent.click(page.getByRole("link", { name: orchardRow.label }), {
    modifiers: ["Shift"],
  });
  await expect.element(bar.getByText("3 Films picked")).toBeVisible();
  await expect.element(page.getByRole("checkbox", { name: "Pick every shown Work" })).toBeChecked();
});
