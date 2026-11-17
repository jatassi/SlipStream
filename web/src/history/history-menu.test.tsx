import { HttpResponse, type RequestHandler, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { problem } from "../test/handlers";
import { anEntry, historyServer } from "../test/history";
import { cinema } from "../test/library";
import { historyKeep, housekeepingHandler } from "../test/notifications";
import { renderApp } from "../test/render-app";

const day = 86_400;

function openMenu(extra: RequestHandler[] = []) {
  const server = historyServer([anEntry(1, new Date().toISOString())]);
  const patched: unknown[] = [];
  renderApp({
    path: "/activity/history",
    handlers: [
      ...extra,
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
      housekeepingHandler,
      http.patch("*/api/v1/tasks/housekeeping", async ({ request }) => {
        patched.push(await request.json());
        return HttpResponse.json({});
      }),
      ...server.handlers,
    ],
  });
  return { server, patched };
}

const choose = async (item: string) => {
  await userEvent.click(page.getByRole("button", { name: "History options" }));
  await userEvent.click(page.getByRole("menuitem", { name: item }));
};

test("Keep history for edits Housekeeping's one stored value, refusing less than its floor", async () => {
  const { patched } = openMenu();
  await choose("Keep history for…");
  const dialog = page.getByRole("dialog", { name: "Keep history for" });
  const field = dialog.getByRole("textbox", { name: "Keep history for" });
  await expect.element(field).toHaveValue("90");
  await expect.element(dialog.getByText(historyKeep.description ?? "")).toBeVisible();
  await userEvent.fill(field, "20");
  await userEvent.click(dialog.getByRole("button", { name: "Save" }));
  await expect.element(dialog.getByRole("alert")).toBeVisible();
  expect(patched).toEqual([]);
  await userEvent.fill(field, "120");
  await userEvent.click(dialog.getByRole("button", { name: "Save" }));
  await expect.poll(() => patched).toEqual([{ tuning: { "history.keepFor": 120 * day } }]);
});

test("Clear History says what it deletes and keeps, then clears and says so", async () => {
  const { server } = openMenu();
  await choose("Clear History…");
  const dialog = page.getByRole("dialog", { name: "Clear History?" });
  await expect
    .element(dialog)
    .toMatchTextContent(
      "Every History entry is deleted, and the bell's list is emptied too. A message that is still being sent stays until it's sent. Grabs, Imports and files aren't deleted. This can't be undone.",
    );
  await userEvent.click(dialog.getByRole("button", { name: "Clear History" }));
  await expect.element(page.getByText("Clearing History.")).toBeVisible();
  expect(server.cleared).toBe(1);
});

test("a refused Clear says so with the problem's sentence", async () => {
  openMenu([
    http.delete("*/api/v1/history", () =>
      problem(
        409,
        "not_controller",
        "This Node isn't the Controller",
        "Open the Controller to clear History.",
      ),
    ),
  ]);
  await choose("Clear History…");
  await userEvent.click(page.getByRole("button", { name: "Clear History", exact: true }));
  await expect
    .element(page.getByText("History wasn't cleared. Open the Controller to clear History."))
    .toBeVisible();
});
