import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { actionsServer, torrent, torrentsServer } from "../test/torrents";

const failed = torrent({
  state: "failed",
  severity: "failure",
  tabs: ["problems", "all"],
  name: "Hold.Fast.2019.1080p.WEB-DL",
  done: 1_000_000_000,
  error: "No such file or directory",
  retryable: true,
});
const inspector = () => page.getByRole("complementary");
const notices = () => page.getByRole("region", { name: "Notifications" });

test("a seed that failed offers Retry, which checks its files", async () => {
  const actions = actionsServer();
  renderApp({
    path: `/activity/torrents/${failed.id}?tab=all`,
    handlers: [actions.handler, ...liveServer().handlers, ...torrentsServer([failed]).handlers],
  });
  await expect
    .element(inspector())
    .toMatchTextContent(
      /The torrent client stopped it with the error "No such file or directory"\. Retry it to check its files\./u,
    );
  await userEvent.click(inspector().getByRole("button", { name: "Retry" }));
  await expect
    .element(notices().getByText("Checking the files of Hold.Fast.2019.1080p.WEB-DL."))
    .toBeVisible();
  expect(actions.bodies).toEqual([{ action: "retry", ids: [failed.id] }]);
});
