import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { getListHistoryMockHandler } from "../api/gen/query/history/history.msw";
import { entriesFrom, historyServer } from "../test/history";
import { orchardId } from "../test/library";
import { renderApp } from "../test/render-app";
import { workServer } from "../test/work-server";

const section = () => page.getByRole("region", { name: "History" });

test("a Work page lists its ten newest History entries and links to all of them", async () => {
  const history = historyServer(entriesFrom(30, 30, new Date()));
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [...history.handlers, ...workServer().handlers],
  });
  await expect.poll(() => section().getByRole("listitem").all().length).toBe(10);
  await expect
    .element(section().getByRole("link", { name: "See all in History" }))
    .toHaveAttribute("href", `/activity/history?work=${orchardId}`);
  expect(history.asked.at(-1)?.get("work")).toBe(orchardId);
  expect(history.asked.at(-1)?.get("limit")).toBe("10");
});

test("a Work with no entries says so in its Content Type's noun", async () => {
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [getListHistoryMockHandler({ entries: [], next: null }), ...workServer().handlers],
  });
  await expect.element(section()).toMatchTextContent(/This film has no History entries yet\./u);
});
