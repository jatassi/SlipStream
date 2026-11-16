import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { orchardRow } from "../test/library";
import { cinemaLibrary, ember } from "../test/library-page";
import { renderApp } from "../test/render-app";
import { wantedServer } from "../test/wanted";

const bar = page.getByRole("toolbar", { name: "Picked Works" });

test("bulk Search queues the picked Works in the bulk lane and says how many", async () => {
  const wanted = wantedServer(undefined, {
    several: { queued: 2, merged: 0, leftGone: 0, works: [] },
  });
  renderApp({ path: "/library", handlers: [...wanted.handlers, ...cinemaLibrary().handlers] });
  await userEvent.click(page.getByRole("button", { name: "Library options" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Select" }));
  await userEvent.click(page.getByRole("checkbox", { name: `Pick ${ember.label}` }));
  await userEvent.click(page.getByRole("checkbox", { name: `Pick ${orchardRow.label}` }));
  await expect.element(bar.getByText("2 Films picked")).toBeVisible();

  await userEvent.click(bar.getByRole("button", { name: "Search" }));
  await expect.element(page.getByText("Queued 2 films for search.")).toBeVisible();
  expect(wanted.searched).toEqual([{ workIds: [ember.id, orchardRow.id] }]);
});
