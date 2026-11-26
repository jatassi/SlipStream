import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getGetTaskQueryKey, getListTasksQueryKey } from "../api/gen/query/tasks/tasks";
import { seeded } from "../test/live-server";
import { mediaServersWorld } from "../test/media-server-writes";
import { home, nodesAre } from "../test/nodes";
import { renderApp } from "../test/render-app";

const path = "/settings/media-servers";

const labels = [
  "Wait before sending",
  "Check sessions every",
  "Mark unhealthy after",
  "Keep play history for",
  "Renew Plex sign-in every",
  "Renew when the sign-in expires within",
];

async function openPanel() {
  const world = mediaServersWorld();
  renderApp({ path, handlers: [nodesAre(home), ...world.handlers] });
  const panel = page.getByRole("region", { name: "Sending and sessions" });
  await userEvent.click(panel.getByRole("button", { name: "Details" }));
  return { world, panel };
}

test("the settings lead with what they add up to and save each value on its own", async () => {
  const { world, panel } = await openPanel();
  await expect
    .element(
      panel.getByText(
        "SlipStream sends a folder's changes 10 s after its last change, checks what's playing every 15 s, keeps play history for 30 days and renews the Plex sign-in every 1 day. The defaults suit most servers.",
      ),
    )
    .toBeVisible();
  await Promise.all(
    labels.map((label) =>
      expect.element(panel.getByRole("textbox", { name: label })).toBeVisible(),
    ),
  );
  await userEvent.fill(panel.getByRole("textbox", { name: "Wait before sending" }), "30");
  await userEvent.click(panel.getByRole("button", { name: "Save" }));
  await expect
    .element(panel.getByText("SlipStream sends a folder's changes 30 s after", { exact: false }))
    .toBeVisible();
  expect(world.writes.at(-1)).toEqual({
    path: "settings",
    body: { values: { "mediaServers.waitBeforeSending": 30 } },
  });
});

test("a renewal margin no longer than the renewal schedule is refused on its field", async () => {
  const { world, panel } = await openPanel();
  const margin = panel.getByRole("textbox", { name: "Renew when the sign-in expires within" });
  await userEvent.fill(margin, "1");
  await userEvent.click(panel.getByRole("button", { name: "Save" }));
  await expect
    .element(
      panel
        .getByRole("alert")
        .getByText(
          '"Renew when the sign-in expires within" must be longer than "Renew Plex sign-in every", which is 1 day. Raise the first or lower the second.',
        ),
    )
    .toBeVisible();
  await userEvent.fill(margin, "2");
  await userEvent.click(panel.getByRole("button", { name: "Save" }));
  await expect.element(panel.getByRole("button", { name: "Reset to default" })).toBeVisible();
  expect(panel.getByRole("alert").query()).toBeNull();
  expect(world.writes.at(-1)).toEqual({
    path: "settings",
    body: { values: { "mediaServers.renewMargin": 172_800 } },
  });
});

test("saving the renewal schedule makes the Tasks page read the renewal task again", async () => {
  const world = mediaServersWorld();
  const app = renderApp({ path, handlers: [nodesAre(home), ...world.handlers] });
  const panel = page.getByRole("region", { name: "Sending and sessions" });
  await userEvent.click(panel.getByRole("button", { name: "Details" }));
  const invalidated = seeded(app, {
    tasks: getListTasksQueryKey(),
    renewal: getGetTaskQueryKey("plex-sign-in"),
  });
  await userEvent.fill(panel.getByRole("textbox", { name: "Wait before sending" }), "30");
  await userEvent.click(panel.getByRole("button", { name: "Save" }));
  await expect.element(panel.getByText("30 s after", { exact: false })).toBeVisible();
  expect(invalidated()).toEqual({ tasks: false, renewal: false });

  await userEvent.fill(panel.getByRole("textbox", { name: "Renew Plex sign-in every" }), "2");
  await userEvent.click(panel.getByRole("button", { name: "Save" }));
  await expect.poll(invalidated).toEqual({ tasks: true, renewal: true });
});
