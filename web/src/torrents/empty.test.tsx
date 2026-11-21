import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { torrent, torrentsServer } from "../test/torrents";

const table = () => page.getByRole("table", { name: "Torrents" });

test("an empty tab names the tab that has torrents and switches to it", async () => {
  const rows = Array.from({ length: 16 }, () =>
    torrent({ state: "seeding", tabs: ["seeding", "all"] }),
  );
  const app = renderApp({
    path: "/activity/torrents",
    handlers: [...liveServer().handlers, ...torrentsServer(rows).handlers],
  });
  await expect
    .element(page.getByText("Nothing is downloading. 16 torrents are seeding."))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Show Seeding" }));
  await expect.poll(() => table().getByRole("row").all()).toHaveLength(17);
  await expect.poll(() => app.router.state.location.search).toEqual({ tab: "seeding" });
});

test("an empty tab with a filter set offers to clear it", async () => {
  renderApp({
    path: "/activity/torrents?tab=seeding&q=nothing",
    handlers: [
      ...liveServer().handlers,
      ...torrentsServer([torrent({ state: "seeding", tabs: ["seeding", "all"] })]).handlers,
    ],
  });
  await expect.element(page.getByText("No torrents match these filters.")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Clear filters" }));
  await expect.poll(() => table().getByRole("row").all()).toHaveLength(2);
});
