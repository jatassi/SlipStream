import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../../../api/gen/query/works/works.msw";
import { cinema, orchardId } from "../../../test/library";
import { cinemaLibrary, ember, harvest } from "../../../test/library-page";
import { renderApp } from "../../../test/render-app";
import { asPhone } from "../../../test/viewport";

const card = (label: string) => page.getByRole("listitem").filter({ hasText: label });

test("/ opens the Library", async () => {
  const { router } = renderApp({
    path: "/",
    handlers: [getListContentTypesMockHandler({ contentTypes: [cinema] })],
  });
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
  expect(router.state.location.href).toBe("/library");
});

test("the Library draws each Work as a poster under its letter, and adds a Work", async () => {
  const { router } = renderApp({ path: "/library", handlers: cinemaLibrary().handlers });
  await expect.element(page.getByRole("heading", { name: "Q 1" })).toBeVisible();
  expect(
    page
      .getByRole("heading", { level: 2 })
      .elements()
      .map((heading) => heading.textContent),
  ).toEqual(["E 1", "O 1", "Q 1"]);
  await expect
    .element(page.getByRole("link", { name: "The Quiet Orchard" }))
    .toHaveAttribute("href", `/works/${orchardId}`);
  await expect.element(card("The Quiet Orchard")).toHaveTextContent("The Quiet Orchard2021Missing");
  await expect.element(card(harvest.label)).toHaveTextContent("Orchard Harvest2019Met");
  await expect.element(card(ember.label)).toHaveTextContent("Ember Lane2024Failed 1Not monitored");

  await userEvent.click(page.getByRole("banner").getByRole("button", { name: "Add" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Add a Film" }));
  await expect.element(page.getByRole("heading", { name: "Add a Film" })).toBeVisible();
  expect(router.state.location.href).toBe("/add?contentType=cinema");
});

test("an empty Library offers to add a Work of each enabled type", async () => {
  const { router } = renderApp({ path: "/library", handlers: cinemaLibrary([]).handlers });
  await expect
    .element(page.getByText("Nothing in the library yet. Add your first Work to start."))
    .toBeVisible();
  await userEvent.click(page.getByRole("main").getByRole("button", { name: "Add a Film" }));
  await expect.element(page.getByRole("heading", { name: "Add a Film" })).toBeVisible();
  expect(router.state.location.href).toBe("/add?contentType=cinema");
});

test("an empty Library with every Content Type off offers nothing to add", async () => {
  renderApp({
    path: "/library",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [{ ...cinema, enabled: false }] }),
      ...cinemaLibrary([]).handlers,
    ],
  });
  await expect
    .element(page.getByText("Nothing in the library yet. Turn on a Content Type to add Works."))
    .toBeVisible();
  await expect.element(page.getByRole("button", { name: "Add" })).not.toBeInTheDocument();
});

test("a Content Type's empty Library offers to add one", async () => {
  const { router } = renderApp({ path: "/library/cinema", handlers: cinemaLibrary([]).handlers });
  await expect.element(page.getByText("No Films in the library yet.")).toBeVisible();
  await userEvent.click(page.getByRole("main").getByRole("button", { name: "Add a Film" }));
  await expect.element(page.getByRole("heading", { name: "Add a Film" })).toBeVisible();
  expect(router.state.location.href).toBe("/add?contentType=cinema");
});

test("on a phone the posters stand two to a row, 16px from each edge", async () => {
  await asPhone();
  renderApp({ path: "/library", handlers: cinemaLibrary().handlers });
  await expect.element(page.getByRole("link", { name: "The Quiet Orchard" })).toBeVisible();
  const rows = page
    .getByRole("main")
    .getByRole("list")
    .filter({ has: page.getByRole("link") })
    .elements();
  const [first] = rows;
  const posters = first === undefined ? [] : [...first.children];
  expect(rows).toHaveLength(3);
  expect(posters.map((poster) => poster.getBoundingClientRect().left)).toEqual([16]);
  expect(first?.getBoundingClientRect().right).toBe(374);
});
