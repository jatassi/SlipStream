import { http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { problem } from "../test/handlers";
import { orchardId } from "../test/library";
import { renderApp } from "../test/render-app";
import { aWorkSearching } from "../test/searching";
import { workServer } from "../test/work-server";

test("a press with nothing to search for says so plainly, not as a failure", async () => {
  const nothing =
    "Nothing to search for. Every monitored, released file meets its Target or is downloading.";
  const server = workServer({ queue: aWorkSearching({ groups: [] }) });
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [
      http.post("*/api/v1/works/:workId/search", () => problem(409, "nothing_wanted", nothing)),
      ...server.handlers,
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Search", exact: true }));
  const notifications = page.getByRole("region", { name: "Messages" });
  await expect.element(notifications.getByText(nothing)).toBeVisible();
  expect(notifications.getByText("Failed", { exact: true }).elements()).toHaveLength(0);
});
