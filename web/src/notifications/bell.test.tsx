import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { notificationsWorld } from "../test/notification-writes";
import { feedRow } from "../test/notifications";
import { renderApp } from "../test/render-app";

const rows = (count: number, unread: number) =>
  Array.from({ length: count }, (_, at) => feedRow(count - at, at < unread));

test("the bell names its unread count, and 99+ from 100", async () => {
  renderApp({ path: "/", handlers: notificationsWorld({ feed: rows(3, 3), unread: 3 }).handlers });
  await expect.element(page.getByRole("button", { name: "Notifications, 3 unread" })).toBeVisible();
});

test("past 99 unread the bell reads 99+", async () => {
  renderApp({
    path: "/",
    handlers: notificationsWorld({ feed: rows(3, 3), unread: 100 }).handlers,
  });
  await expect
    .element(page.getByRole("button", { name: "Notifications, 99+ unread" }))
    .toBeVisible();
});

test("opening the bell lists the newest rows and moves the cursor to the newest", async () => {
  const world = notificationsWorld({ feed: rows(3, 2), unread: 2 });
  renderApp({ path: "/", handlers: world.handlers });
  await userEvent.click(page.getByRole("button", { name: "Notifications, 2 unread" }));
  const list = page
    .getByRole("dialog", { name: "Notifications" })
    .getByRole("list", { name: "Notifications" });
  await expect.element(list.getByRole("link", { name: /Imported Heat 3/u })).toBeVisible();
  await expect.element(list.getByText("Unread")).toHaveLength(2);
  await expect.poll(() => world.writes).toEqual([{ path: "seen", body: { through: 3 } }]);
  await userEvent.keyboard("{Escape}");
  await expect
    .element(page.getByRole("button", { name: "Notifications", exact: true }))
    .toBeVisible();
});

test("the feed pages older rows on Show older notifications", async () => {
  const world = notificationsWorld({ feed: rows(35, 0) });
  renderApp({ path: "/", handlers: world.handlers });
  await userEvent.click(page.getByRole("button", { name: "Notifications", exact: true }));
  const dialog = page.getByRole("dialog", { name: "Notifications" });
  await expect.element(dialog.getByText("Imported Heat 6", { exact: true })).toBeVisible();
  expect(dialog.getByText("Imported Heat 5", { exact: true }).query()).toBeNull();
  await userEvent.click(dialog.getByRole("button", { name: "Show older notifications" }));
  await expect.element(dialog.getByText("Imported Heat 1", { exact: true })).toBeVisible();
  expect(dialog.getByRole("button", { name: "Show older notifications" }).query()).toBeNull();
});

test("an empty bell says how long it keeps notifications", async () => {
  renderApp({ path: "/", handlers: notificationsWorld().handlers });
  await userEvent.click(page.getByRole("button", { name: "Notifications", exact: true }));
  await expect.element(page.getByText("No notifications in the last 30 days.")).toBeVisible();
});
