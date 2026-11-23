import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { mediaServersWorld } from "../../test/media-server-writes";
import { denJellyfin, livingRoomPlex } from "../../test/media-servers";
import { renderApp } from "../../test/render-app";
import { asPhone } from "../../test/viewport";

const path = `/settings/media-servers/${denJellyfin.id}`;

const sessionsAre = (body: object) =>
  http.get(`/api/v1/media-servers/${denJellyfin.id}/sessions`, () => HttpResponse.json(body));

test("a server's page leads with its state and what's playing, and keeps the machinery in Details", async () => {
  const world = mediaServersWorld();
  renderApp({ path, handlers: world.handlers });
  await expect.element(page.getByRole("heading", { name: "Den Jellyfin" })).toBeVisible();
  await expect.element(page.getByText(/^Den Jellyfin is up to date/u)).toBeVisible();
  const tabs = page.getByRole("navigation", { name: "Den Jellyfin" });
  await Promise.all(
    ["Now playing", "Changes", "Path mappings", "Settings", "Details"].map((tab) =>
      expect.element(tabs.getByRole("link", { name: tab })).toBeVisible(),
    ),
  );
  const playing = page.getByRole("list", { name: "Now playing on Den Jellyfin" });
  await expect
    .element(playing.getByText("The Quiet Orchard (2021) Bluray-1080p.mkv"))
    .toBeVisible();
  await expect
    .element(
      playing.getByText("grace on Living room TV · Jellyfin for Android TV", { exact: false }),
    )
    .toBeVisible();
  await userEvent.hover(playing.getByRole("button", { name: "Hardware transcode" }));
  await expect
    .element(page.getByRole("dialog", { name: "Hardware transcode" }))
    .toMatchTextContent(/graphics hardware/u);
  await expect.element(playing.getByText(/^Transcoding because .+container/u)).toBeVisible();
  await userEvent.click(playing.getByRole("button", { name: "Details" }));
  await expect.element(playing.getByText(/Den Jellyfin.+ContainerNotSupported/u)).toBeVisible();

  await userEvent.click(tabs.getByRole("link", { name: "Details" }));
  await expect.element(page.getByText("Jellyfin 12.2.0")).toBeVisible();
  await expect.element(page.getByText("The Controller reaches it")).toBeVisible();

  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await expect.element(page.getByText(/^home connected to Den Jellyfin 12\.2\.0/u)).toBeVisible();
  await expect.element(page.getByText(/^The Controller sends changes/u)).toBeVisible();
  expect(world.writes).toEqual([{ path: "test", body: null }]);
});

test("Now playing says when nobody is playing", async () => {
  renderApp({
    path,
    handlers: [sessionsAre({ reachable: true, sessions: [] }), ...mediaServersWorld().handlers],
  });
  await expect.element(page.getByText(/^Nobody is playing .+Den Jellyfin/u)).toBeVisible();
});

test("Now playing can't check sessions while the reaching Node is offline", async () => {
  renderApp({
    path,
    handlers: [sessionsAre({ reachable: false, sessions: [] }), ...mediaServersWorld().handlers],
  });
  await expect.element(page.getByText(/Den Jellyfin while home is offline/u)).toBeVisible();
});

test("the Changes tab says what is failing or waiting, and what each rename lost", async () => {
  renderApp({ path: `${path}/changes`, handlers: mediaServersWorld().handlers });
  await expect
    .element(page.getByText(/^1 change couldn't be sent to Den Jellyfin.+1 more is waiting/u))
    .toBeVisible();
  const changes = page.getByRole("list", { name: "Changes on Den Jellyfin" });
  const row = (name: string) => changes.getByRole("listitem").filter({ hasText: name });
  await expect.element(row("Night Train (2018) DVD.mkv").getByText("Failing")).toBeVisible();
  await expect.element(row("Night Train (2018) DVD.mkv").getByText(/status 500/u)).toBeVisible();
  await expect.element(row("The Long Field (2019)").getByText("Waiting")).toBeVisible();
  const rename = row("The Quiet Orchard (2021) Bluray-1080p.mkv");
  await expect.element(rename.getByText("Lost data")).toBeVisible();
  await expect.element(rename.getByText(/play count, last played/u)).toBeVisible();
  await expect
    .element(rename.getByText("The Quiet Orchard (2021)/The Quiet Orchard.mkv"))
    .toBeVisible();
  await expect.element(row("Sent the new file.").getByText("Sent", { exact: true })).toBeVisible();
});

test("Refresh libraries asks first, then queues the refresh", async () => {
  const world = mediaServersWorld();
  renderApp({ path: `${path}/changes`, handlers: world.handlers });
  await userEvent.click(page.getByRole("button", { name: "Refresh libraries" }));
  const dialog = page.getByRole("dialog", { name: "Refresh every library on Den Jellyfin?" });
  await userEvent.click(dialog.getByRole("button", { name: "Refresh" }));
  await expect.element(page.getByText(/^Queued a refresh of Den Jellyfin/u)).toBeVisible();
  expect(world.writes).toEqual([{ path: "refresh", body: null }]);
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

test("a Plex server's Settings keep its address under Details, as plex.tv listed it", async () => {
  const world = mediaServersWorld([denJellyfin, livingRoomPlex]);
  renderApp({
    path: `/settings/media-servers/${livingRoomPlex.id}/settings`,
    handlers: world.handlers,
  });
  const form = page.getByRole("form", { name: "Settings of Living room Plex" });
  await expect.element(form.getByRole("textbox", { name: "Name" })).toBeVisible();
  expect(form.getByRole("textbox", { name: "Address" }).query()).toBeNull();
  expect(form.getByRole("textbox", { name: "API key" }).query()).toBeNull();
  await userEvent.click(form.getByRole("button", { name: "Details" }));
  const address = form.getByRole("textbox", { name: "Address" });
  await expect.element(address).toHaveValue(livingRoomPlex.address);
  await userEvent.fill(address, "http://192.168.1.30:32400");
  await userEvent.click(form.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Saved Living room Plex")).toBeVisible();
  expect(world.writes).toEqual([{ path: "edit", body: { address: "http://192.168.1.30:32400" } }]);
});

test("on phone the server's top page shows what's playing and lists its other sections as rows", async () => {
  await asPhone();
  renderApp({ path, handlers: mediaServersWorld().handlers });
  const rows = page.getByRole("list", { name: "Den Jellyfin" });
  await expect
    .element(page.getByRole("list", { name: "Now playing on Den Jellyfin" }))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Hardware transcode" }));
  await expect
    .element(page.getByRole("dialog", { name: "Hardware transcode" }))
    .toMatchTextContent(/graphics hardware/u);
  await userEvent.keyboard("{Escape}");
  await Promise.all(
    ["Changes", "Path mappings", "Settings", "Details"].map((name) =>
      expect.element(rows.getByRole("link", { name })).toBeVisible(),
    ),
  );
  await userEvent.click(rows.getByRole("link", { name: "Path mappings" }));
  await expect
    .element(page.getByRole("form", { name: "Path mappings on Den Jellyfin" }))
    .toBeVisible();
});
