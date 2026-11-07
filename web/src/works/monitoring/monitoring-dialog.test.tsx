import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { WorksMonitoringChange } from "../../api/gen/model";
import { anthologyId, longShelf, longShelfCells } from "../../test/anthology";
import { orchardId } from "../../test/library";
import { longShelfCounts, monitoringServer } from "../../test/monitoring";
import { renderApp } from "../../test/render-app";
import { workServer } from "../../test/work-server";

const monitoredNow = 1176;

/** Every Story of The Long Shelf but the Extras is monitored, so a preset stops the rest and Include starts the Extras. */
const moves = ({ preset, includeAsides }: WorksMonitoringChange) => ({
  start: includeAsides ? longShelfCounts.asides : 0,
  stop:
    monitoredNow - (longShelfCounts.choices.find((each) => each.preset === preset)?.monitors ?? 0),
});

const dialog = page.getByRole("dialog", { name: "Set monitoring for The Long Shelf" });

function openShelf() {
  const monitoring = monitoringServer(moves);
  const server = workServer({ work: longShelf, startCells: longShelfCells });
  renderApp({ path: `/works/${anthologyId}`, handlers: [monitoring.handler, ...server.handlers] });
  return monitoring;
}

test("a Work of many Items has a Monitoring button that counts its monitored Items", async () => {
  openShelf();
  await expect
    .element(page.getByRole("button", { name: "Monitoring 1176 of 1200 stories" }))
    .toBeVisible();
  await expect.element(page.getByRole("switch", { name: "Monitored" })).not.toBeInTheDocument();
});

test("a Work of one Item keeps its Monitored switch", async () => {
  renderApp({ path: `/works/${orchardId}`, handlers: workServer().handlers });
  await expect.element(page.getByRole("switch", { name: "Monitored" })).toBeChecked();
  await expect.element(page.getByRole("button", { name: /^Monitoring/u })).not.toBeInTheDocument();
});

test("the dialog lists each preset with its count and says how many Items start and stop", async () => {
  const monitoring = openShelf();
  await userEvent.click(page.getByRole("button", { name: "Monitoring 1176 of 1200 stories" }));

  const choices = dialog.getByRole("radiogroup", { name: "Stories to monitor" });
  await expect.element(choices.getByRole("radio", { name: "All · 1176 stories" })).toBeChecked();
  expect(
    choices
      .getByRole("radio")
      .all()
      .map((radio) => radio.element().closest("label")?.textContent),
  ).toEqual([
    "All · 1176 stories",
    "Future · 0 stories",
    "Missing · 1176 stories",
    "Existing · 0 stories",
    "First volume · 49 stories",
    "Latest volume · 49 stories",
    "None · 0 stories",
  ]);
  await expect.element(dialog.getByText("No stories change.")).toBeVisible();
  await expect.element(dialog.getByRole("switch", { name: "Include extras" })).not.toBeChecked();
  await expect.element(dialog.getByRole("switch", { name: "Monitor new stories" })).toBeChecked();

  await userEvent.click(choices.getByRole("radio", { name: "First volume · 49 stories" }));
  await expect.element(dialog.getByText("1127 stories stop being monitored.")).toBeVisible();
  await expect
    .element(dialog.getByRole("switch", { name: "Monitor new stories" }))
    .not.toBeChecked();

  await userEvent.click(dialog.getByRole("switch", { name: "Include extras" }));
  await expect
    .element(choices.getByRole("radio", { name: "First volume · 73 stories" }))
    .toBeChecked();
  await expect
    .element(dialog.getByText("24 stories start being monitored and 1127 stop."))
    .toBeVisible();

  await userEvent.click(dialog.getByRole("button", { name: "Apply" }));
  await expect.element(page.getByText("Changed the monitoring of The Long Shelf.")).toBeVisible();
  await expect.element(dialog).not.toBeInTheDocument();
  expect(monitoring.sent.at(-1)).toEqual({
    workIds: [anthologyId],
    preset: "first",
    includeAsides: true,
    preview: false,
  });
  expect(monitoring.sent.filter(({ preview }) => !preview)).toHaveLength(1);
});

test("Monitor new follows the preset until the Admin sets it, and a new preset follows again", async () => {
  const monitoring = openShelf();
  await userEvent.click(page.getByRole("button", { name: "Monitoring 1176 of 1200 stories" }));
  const choices = dialog.getByRole("radiogroup", { name: "Stories to monitor" });
  await userEvent.click(choices.getByRole("radio", { name: "Future · 0 stories" }));
  await userEvent.click(dialog.getByRole("switch", { name: "Monitor new stories" }));
  await expect
    .element(dialog.getByRole("switch", { name: "Monitor new stories" }))
    .not.toBeChecked();
  await expect.poll(() => monitoring.sent.at(-1)?.monitorNew).toBe(false);

  await userEvent.click(choices.getByRole("radio", { name: "Latest volume · 49 stories" }));
  await expect.element(dialog.getByRole("switch", { name: "Monitor new stories" })).toBeChecked();
  await userEvent.click(dialog.getByRole("button", { name: "Apply" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(monitoring.sent.at(-1)).toEqual({
    workIds: [anthologyId],
    preset: "latest",
    includeAsides: false,
    preview: false,
  });
});
