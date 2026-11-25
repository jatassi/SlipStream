import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { anEntry, entriesFrom, historyServer } from "../test/history";
import { cinema } from "../test/library";
import { liveServer, synced } from "../test/live-server";
import { renderApp } from "../test/render-app";

const rows = () => page.getByRole("main").getByRole("listitem");

test("history.changed reads the first page once and puts the new entries on top of every page loaded", async () => {
  const live = liveServer();
  const history = historyServer(entriesFrom(80, 80, new Date(Date.now() - 60_000)));
  const app = renderApp({
    path: "/activity/history",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
      ...live.handlers,
      ...history.handlers,
    ],
  });
  await synced(app);
  await userEvent.click(page.getByRole("button", { name: "Load more" }));
  await expect.poll(() => rows().all().length).toBe(80);
  await expect.poll(() => app.queryClient.isFetching()).toBe(0);
  const [lists, summaries] = [history.asked.length, history.summaries.length];

  history.record(anEntry(81, new Date().toISOString()));
  history.record(anEntry(82, new Date().toISOString()));
  expect({ clients: live.clients.length, closed: live.closed }).toEqual({ clients: 1, closed: [] });
  live.clients[0]?.send({ event: "history.changed", data: {} });

  await expect.poll(() => rows().all().length).toBe(82);
  await expect.element(rows().nth(0)).toMatchTextContent(/Orchard 82/u);
  await expect.poll(() => history.summaries.length).toBe(summaries + 1);
  expect(history.asked.slice(lists).map((asked) => asked.get("before"))).toEqual([null]);
});

test("a Clear between events leaves only what the first page now holds", async () => {
  const live = liveServer();
  const history = historyServer(entriesFrom(3, 3, new Date()));
  const app = renderApp({
    path: "/activity/history",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
      ...live.handlers,
      ...history.handlers,
    ],
  });
  await synced(app);
  await expect.poll(() => rows().all().length).toBe(3);
  await userEvent.click(page.getByRole("button", { name: "History options" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Clear History…" }));
  await userEvent.click(page.getByRole("button", { name: "Clear History", exact: true }));
  await expect.poll(() => history.cleared).toBe(1);
  history.record(anEntry(4, new Date().toISOString()));
  live.clients[0]?.send({ event: "history.changed", data: {} });
  await expect.poll(() => rows().all().length).toBe(1);
  await expect.element(rows().nth(0)).toMatchTextContent(/Orchard 4/u);
});
