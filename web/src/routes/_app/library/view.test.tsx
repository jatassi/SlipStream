import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getSearchWorksMockHandler } from "../../../api/gen/query/works/works.msw";
import { readPref, writePref } from "../../../shell/prefs";
import { orchardRow } from "../../../test/library";
import { cinemaLibrary, ember, harvest } from "../../../test/library-page";
import { renderApp } from "../../../test/render-app";

const headings = () =>
  page
    .getByRole("main")
    .getByRole("heading", { level: 2 })
    .elements()
    .map((heading) => heading.textContent);

const shownWorks = () =>
  page
    .getByRole("main")
    .getByRole("link")
    .elements()
    .map((link) => link.textContent);

test("sorting by Profile sections the Library by Profile, and Reverse flips the sections", async () => {
  const { router } = renderApp({ path: "/library", handlers: cinemaLibrary().handlers });
  await userEvent.click(page.getByRole("button", { name: "Sort: Title" }));
  await userEvent.click(page.getByRole("menuitemcheckbox", { name: "Profile" }));
  await userEvent.keyboard("{Escape}");
  await expect.element(page.getByRole("heading", { name: "Ultra 1" })).toBeVisible();
  expect(headings()).toEqual(["Films 2", "Ultra 1"]);
  expect(shownWorks()).toEqual([ember.label, orchardRow.label, harvest.label]);
  expect(router.state.location.href).toBe("/library?sort=profile");

  await userEvent.click(page.getByRole("button", { name: "Sort: Profile" }));
  await userEvent.click(page.getByRole("menuitemcheckbox", { name: "Reverse" }));
  await userEvent.keyboard("{Escape}");
  await expect.element(page.getByRole("button", { name: "Sort: Profile, reversed" })).toBeVisible();
  expect(headings()).toEqual(["Ultra 1", "Films 2"]);
  expect(router.state.location.href).toBe("/library?sort=profile&rev=1");
});

test("a view changed in the toolbar goes into the address and is kept for the next visit", async () => {
  const { router } = renderApp({ path: "/library", handlers: cinemaLibrary().handlers });
  await userEvent.click(page.getByRole("button", { name: "Library options" }));
  await userEvent.click(page.getByRole("menuitemcheckbox", { name: "Table" }));
  await userEvent.keyboard("{Escape}");
  const table = page.getByRole("table", { name: "Films" });
  await expect.element(table).toBeVisible();
  await expect.poll(() => table.getByRole("row").elements().length).toBe(4);
  expect(router.state.location.href).toBe("/library?view=table");
  expect(readPref("libraryViews")).toEqual({ all: { view: "table" } });

  await userEvent.click(page.getByRole("button", { name: "Library options" }));
  await userEvent.click(page.getByRole("menuitemcheckbox", { name: "Size on disk" }));
  await userEvent.keyboard("{Escape}");
  await expect
    .element(table.getByRole("columnheader", { name: "Size on disk" }))
    .not.toBeInTheDocument();
  expect(router.state.location.href).toBe(
    "/library?view=table&cols=status%2Cprofile%2Cadded%2Cfield.year%2Cfield.runtime",
  );
});

test("a visit naming no view opens on the one last used, keeping its search", async () => {
  writePref("libraryViews", { all: { view: "table", sort: "profile" }, cinema: { size: "large" } });
  const { router } = renderApp({
    path: "/library?q=harvest",
    handlers: [
      getSearchWorksMockHandler({ rows: [], workIds: [harvest.id] }),
      ...cinemaLibrary().handlers,
    ],
  });
  await expect.element(page.getByRole("table", { name: "Films" })).toBeVisible();
  expect(router.state.location.href).toBe("/library?view=table&sort=profile&q=harvest");
  await expect.poll(shownWorks).toEqual([harvest.label]);
  await expect.element(page.getByText("1 of 3 Films")).toBeVisible();
});

test("a filter keeps the Works of any status picked, and Clear filters brings every one back", async () => {
  const { router } = renderApp({ path: "/library", handlers: cinemaLibrary().handlers });
  await userEvent.click(page.getByRole("button", { name: "Filter" }));
  await userEvent.click(page.getByRole("checkbox", { name: "Missing (1)" }));
  await userEvent.click(page.getByRole("checkbox", { name: "Failed (1)" }));
  await userEvent.keyboard("{Escape}");
  await expect.element(page.getByText("2 of 3 Films")).toBeVisible();
  expect(shownWorks()).toEqual([ember.label, orchardRow.label]);
  expect(router.state.location.href).toBe("/library?status=missing%2Cfailed");

  await userEvent.click(page.getByRole("button", { name: "Filter (1)" }));
  await userEvent.click(page.getByRole("checkbox", { name: "Not monitored (1)" }));
  await userEvent.keyboard("{Escape}");
  await expect.element(page.getByText("1 of 3 Films")).toBeVisible();
  expect(shownWorks()).toEqual([ember.label]);

  await userEvent.click(page.getByRole("main").getByRole("button", { name: "Clear filters" }));
  await expect.poll(shownWorks).toEqual([ember.label, harvest.label, orchardRow.label]);
  expect(router.state.location.href).toBe("/library");
});

test("the arrow keys move between posters and Space picks the one in focus", async () => {
  renderApp({ path: "/library", handlers: cinemaLibrary().handlers });
  const emberLink = page.getByRole("link", { name: ember.label });
  await expect.element(emberLink).toHaveAttribute("tabindex", "0");
  emberLink.element().focus();
  await userEvent.keyboard("{ArrowRight}");
  await expect.element(page.getByRole("link", { name: harvest.label })).toHaveFocus();
  await userEvent.keyboard("{End}");
  await expect.element(page.getByRole("link", { name: orchardRow.label })).toHaveFocus();
  await userEvent.keyboard("{ArrowUp}");
  await expect.element(page.getByRole("link", { name: harvest.label })).toHaveFocus();
  await userEvent.keyboard(" ");
  const bar = page.getByRole("toolbar", { name: "Picked Works" });
  await expect.element(bar.getByText("1 Film picked")).toBeVisible();
  await expect.element(page.getByRole("checkbox", { name: `Pick ${harvest.label}` })).toBeChecked();
});
