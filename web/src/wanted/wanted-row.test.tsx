import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { anthologies } from "../test/anthology";
import { renderApp } from "../test/render-app";
import { aWantedPage, goneOnly, missingRange, wantedServer } from "../test/wanted";

function openRows(server = wantedServer([aWantedPage([missingRange])])) {
  const app = renderApp({
    path: "/library/wanted",
    handlers: [getListContentTypesMockHandler({ contentTypes: [anthologies] }), ...server.handlers],
  });
  return { app, server };
}

const rowOf = (label: string) => page.getByRole("row").filter({ hasText: label });

test("a row shows the Work, its Items, why, the last search and the next", async () => {
  openRows(wantedServer([aWantedPage([missingRange, goneOnly])]));
  const shelf = rowOf("The Long Shelf");
  await expect.element(shelf).toBeVisible();
  expect(
    shelf
      .getByRole("cell")
      .elements()
      .slice(0, 4)
      .map((cell) => cell.textContent),
  ).toEqual([
    "The Long Shelf",
    "Story V02S01 to Story V02S08",
    "Missing, 3 upgrades",
    expect.stringMatching(/^Found nothing /u),
  ]);
  await expect
    .element(rowOf("The Quiet Orchard"))
    .toMatchTextContent(/File gone.*Only when you search/u);
});

test("a press on a Work already searching says so", async () => {
  openRows(
    wantedServer([aWantedPage([missingRange])], {
      one: { groups: [{ key: "l:1", label: "volume 2", itemIds: ["i"], state: "merged" }] },
    }),
  );
  await userEvent.click(page.getByRole("button", { name: "Search The Long Shelf" }));
  await expect.element(page.getByText("Already searching The Long Shelf")).toBeVisible();
});

test("the row's menu opens its Work", async () => {
  const { app } = openRows();
  await userEvent.click(page.getByRole("button", { name: "More for The Long Shelf" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Open Work" }));
  await expect
    .poll(() => app.router.state.location.pathname)
    .toBe(`/works/${missingRange.work.id}`);
});
