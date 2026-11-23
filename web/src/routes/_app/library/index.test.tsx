import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../../../api/gen/query/works/works.msw";
import { cinema } from "../../../test/library";
import { cinemaLibrary } from "../../../test/library-page";
import { renderApp } from "../../../test/render-app";

test("an empty Library with every Content Type off offers nothing to add", async () => {
  renderApp({
    path: "/library",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [{ ...cinema, enabled: false }] }),
      ...cinemaLibrary([]).handlers,
    ],
  });
  await expect
    .element(
      page.getByText("Nothing in the library yet. Turn on a kind of media in Settings to start."),
    )
    .toBeVisible();
  await expect.element(page.getByRole("button", { name: "Add" })).not.toBeInTheDocument();
});

test("a Content Type's empty Library offers to add one", async () => {
  const { router } = renderApp({ path: "/library/cinema", handlers: cinemaLibrary([]).handlers });
  await expect.element(page.getByText("No Films in the library yet.")).toBeVisible();
  await userEvent.click(page.getByRole("main").getByRole("button", { name: "Add a film" }));
  await expect.element(page.getByRole("heading", { name: "Add a film" })).toBeVisible();
  expect(router.state.location.href).toBe("/add?contentType=cinema");
});
