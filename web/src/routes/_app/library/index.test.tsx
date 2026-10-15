import { expect, onTestFinished, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import {
  getListContentTypesMockHandler,
  getListWorksMockHandler,
} from "../../../api/gen/query/works/works.msw";
import { cinema, orchard, orchardId } from "../../../test/library";
import { renderApp } from "../../../test/render-app";

const listedOrchard = {
  id: orchardId,
  contentType: "cinema",
  key: "9000001",
  label: "The Quiet Orchard",
  sortLabel: "Quiet Orchard",
  addedAt: "2026-10-12T09:00:00Z",
  refreshedAt: "2026-10-12T09:00:00Z",
  fields: orchard.fields,
};

test("/ opens the Library", async () => {
  const { router } = renderApp({
    path: "/",
    handlers: [getListContentTypesMockHandler({ contentTypes: [cinema] })],
  });
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
  expect(router.state.location.href).toBe("/library");
});

test("the Library lists each Work with its year and adds a Work of an enabled type", async () => {
  const { router } = renderApp({
    path: "/library",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
      getListWorksMockHandler({ works: [listedOrchard] }),
    ],
  });
  await expect.element(page.getByRole("heading", { name: "Library" })).toBeVisible();
  const work = page.getByRole("list", { name: "Works" }).getByRole("listitem");
  await expect.element(work).toHaveTextContent("The Quiet Orchard2021");
  await expect
    .element(work.getByRole("link", { name: "The Quiet Orchard 2021" }))
    .toHaveAttribute("href", `/works/${orchardId}`);

  await userEvent.click(page.getByRole("banner").getByRole("button", { name: "Add" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Add a Film" }));
  await expect.element(page.getByRole("heading", { name: "Add a Film" })).toBeVisible();
  expect(router.state.location.href).toBe("/add?contentType=cinema");
});

test("an empty Library offers to add a Work of each enabled type", async () => {
  const { router } = renderApp({
    path: "/library",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
      getListWorksMockHandler({ works: [] }),
    ],
  });
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
      getListWorksMockHandler({ works: [] }),
    ],
  });
  await expect
    .element(page.getByText("Nothing in the library yet. Turn on a Content Type to add Works."))
    .toBeVisible();
  await expect.element(page.getByRole("button", { name: "Add" })).not.toBeInTheDocument();
});

test("a Content Type's empty Library offers to add one", async () => {
  const { router } = renderApp({
    path: "/library/cinema",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
      getListWorksMockHandler({ works: [] }),
    ],
  });
  await expect.element(page.getByText("No Films in the library yet.")).toBeVisible();
  await userEvent.click(page.getByRole("main").getByRole("button", { name: "Add a Film" }));
  await expect.element(page.getByRole("heading", { name: "Add a Film" })).toBeVisible();
  expect(router.state.location.href).toBe("/add?contentType=cinema");
});

test("on a phone the Library's list sits in an inset group 16px from each edge", async () => {
  await page.viewport(390, 844);
  onTestFinished(async () => {
    await page.viewport(1440, 900);
  });
  renderApp({
    path: "/library",
    handlers: [
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
      getListWorksMockHandler({ works: [listedOrchard] }),
    ],
  });
  const list = page.getByRole("list", { name: "Works" });
  await expect.element(list).toBeVisible();
  const { left, right } = list.element().getBoundingClientRect();
  expect([left, right]).toEqual([16, 374]);
});
