import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { addServer, anthologyCatalog } from "../../test/adding";
import { renderApp } from "../../test/render-app";

const results = () => page.getByRole("list", { name: "Results" }).getByRole("listitem");

test("a result the type's source doesn't list says so in the type's words and offers no Add", async () => {
  renderApp({
    path: "/add?contentType=anthology&query=shelf",
    handlers: addServer({ catalogs: [anthologyCatalog()] }).handlers,
  });

  await expect
    .element(results().last())
    .toHaveTextContent(
      "The Lost ShelfTide BooksFixture source doesn't list this anthology. SlipStream needs Fixture source to keep it up to date.",
    );
  expect(results().last().getByRole("button").query()).toBeNull();
  expect(results().last().getByText("Stories nobody catalogued.").query()).toBeNull();

  await expect
    .element(results().first())
    .toHaveTextContent(
      "The Long ShelfHarbor PressStories from a harbour town, one volume a year.Add",
    );
  await userEvent.click(page.getByRole("button", { name: "Add The Long Shelf" }));
  await expect.element(page.getByRole("dialog", { name: "Add The Long Shelf" })).toBeVisible();
});
