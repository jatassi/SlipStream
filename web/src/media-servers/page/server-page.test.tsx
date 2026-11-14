import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { mediaServersWorld } from "../../test/media-server-writes";
import { denJellyfin } from "../../test/media-servers";
import { renderApp } from "../../test/render-app";
import { asPhone } from "../../test/viewport";

const path = `/settings/media-servers/${denJellyfin.id}`;

test("a server's page leads with its state, tests both routes and keeps the machinery in Details", async () => {
  const world = mediaServersWorld();
  renderApp({ path, handlers: world.handlers });
  await expect.element(page.getByRole("heading", { name: "Den Jellyfin" })).toBeVisible();
  await expect.element(page.getByText("Den Jellyfin is up to date.")).toBeVisible();
  const tabs = page.getByRole("navigation", { name: "Den Jellyfin" });
  await Promise.all(
    ["Details", "Path mappings", "Settings"].map((tab) =>
      expect.element(tabs.getByRole("link", { name: tab })).toBeVisible(),
    ),
  );
  await expect.element(page.getByText("Jellyfin 12.2.0")).toBeVisible();
  await expect.element(page.getByText("The Controller reaches it")).toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await expect.element(page.getByText("home reached Den Jellyfin 12.2.0.")).toBeVisible();
  await expect
    .element(page.getByText("The Controller sends changes while the reaching Node is offline."))
    .toBeVisible();
  expect(world.writes).toEqual([{ path: "test", body: null }]);
});

test("the Settings tab saves only what changed", async () => {
  const world = mediaServersWorld();
  renderApp({ path: `${path}/settings`, handlers: world.handlers });
  const form = page.getByRole("form", { name: "Settings of Den Jellyfin" });
  await expect.element(form.getByText("Saved")).toBeVisible();
  await userEvent.fill(form.getByRole("textbox", { name: "Name" }), "Den");
  await userEvent.click(form.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Saved Den")).toBeVisible();
  expect(world.writes).toEqual([{ path: "edit", body: { name: "Den" } }]);
});

test("on phone the server's top page lists its other sections as rows", async () => {
  await asPhone();
  renderApp({ path, handlers: mediaServersWorld().handlers });
  const rows = page.getByRole("list", { name: "Den Jellyfin" });
  await expect.element(rows.getByRole("link", { name: "Path mappings" })).toBeVisible();
  await expect.element(rows.getByRole("link", { name: "Settings" })).toBeVisible();
  await userEvent.click(rows.getByRole("link", { name: "Path mappings" }));
  await expect
    .element(page.getByRole("form", { name: "Path mappings on Den Jellyfin" }))
    .toBeVisible();
});
