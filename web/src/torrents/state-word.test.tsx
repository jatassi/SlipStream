import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { torrent, torrentsServer } from "../test/torrents";

const unregistered = torrent({
  state: "tracker-error",
  severity: "failure",
  tabs: ["problems", "all"],
  name: "Gone.Release.2020.1080p",
  announce: { host: "beacon.example", status: "unregistered" },
});
const downloading = torrent({
  state: "downloading",
  tabs: ["downloading", "all"],
  name: "Quiet.Orchard.2021.1080p",
});
const row = (name: RegExp) =>
  page.getByRole("table", { name: "Torrents" }).getByRole("row", { name });

test("a state word opens its definition on a press, and the row stays closed", async () => {
  renderApp({
    path: "/activity/torrents?tab=all",
    handlers: [...liveServer().handlers, ...torrentsServer([unregistered, downloading]).handlers],
  });
  await userEvent.click(row(/^Gone\.Release/u).getByRole("button", { name: "Tracker error" }));
  await expect
    .element(page.getByRole("dialog", { name: "Tracker error" }))
    .toHaveTextContent("None of the torrent's trackers is working, so it can't find new peers.");
  await expect.element(page.getByRole("complementary")).not.toBeInTheDocument();
  await expect.element(row(/^Quiet\.Orchard/u).getByText("Downloading")).toBeVisible();
  await expect
    .element(row(/^Quiet\.Orchard/u).getByRole("button", { name: "Downloading" }))
    .not.toBeInTheDocument();
});
