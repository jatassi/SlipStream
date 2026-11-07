import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { cinemaLibrary, harborTales, twoTypeLibrary } from "../test/library-page";
import { renderApp } from "../test/render-app";

const typeSwitch = () => page.getByRole("navigation", { name: "Content Types" });

test.each(["/library", "/library/cinema"])(
  "with one type on, %s has no type switch",
  async (path) => {
    renderApp({ path, handlers: cinemaLibrary().handlers });
    await expect.element(page.getByRole("link", { name: "Ember Lane" })).toBeVisible();
    await expect.element(typeSwitch()).not.toBeInTheDocument();
  },
);

test("the switch offers All and each enabled type in sidebar order, and marks the one shown", async () => {
  const { router } = renderApp({ path: "/library", handlers: twoTypeLibrary() });
  await expect.element(typeSwitch()).toBeVisible();
  expect(
    typeSwitch()
      .getByRole("link")
      .elements()
      .map((link) => [link.textContent, link.getAttribute("href")]),
  ).toEqual([
    ["All", "/library"],
    ["Films", "/library/cinema"],
    ["Anthologies", "/library/anthology"],
  ]);
  await expect
    .element(typeSwitch().getByRole("link", { name: "All" }))
    .toHaveAttribute("aria-current", "page");

  await userEvent.click(typeSwitch().getByRole("link", { name: "Anthologies" }));
  await expect.element(page.getByRole("heading", { level: 1, name: "Anthologies" })).toBeVisible();
  expect(router.state.location.pathname).toBe("/library/anthology");
  await expect
    .element(typeSwitch().getByRole("link", { name: "Anthologies" }))
    .toHaveAttribute("aria-current", "page");
  await expect
    .element(typeSwitch().getByRole("link", { name: "All" }))
    .not.toHaveAttribute("aria-current");
  await expect.element(page.getByRole("link", { name: "Ember Lane" })).not.toBeInTheDocument();
});

test("each segment opens on the view its own Library last used", async () => {
  const { router } = renderApp({ path: "/library", handlers: twoTypeLibrary() });
  await userEvent.click(page.getByRole("button", { name: "Library options" }));
  await userEvent.click(page.getByRole("menuitemcheckbox", { name: "Table" }));
  await userEvent.keyboard("{Escape}");
  await expect.element(page.getByRole("table", { name: "Works" })).toBeVisible();

  await userEvent.click(typeSwitch().getByRole("link", { name: "Anthologies" }));
  await expect.element(page.getByRole("link", { name: harborTales.label })).toBeVisible();
  await expect.element(page.getByRole("table")).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("button", { name: "Sort: Title" }));
  await userEvent.click(page.getByRole("menuitemcheckbox", { name: "Publisher" }));
  await userEvent.keyboard("{Escape}");
  expect(router.state.location.href).toBe("/library/anthology?sort=field.publisher");

  await userEvent.click(typeSwitch().getByRole("link", { name: "All" }));
  await expect.element(page.getByRole("table", { name: "Works" })).toBeVisible();
  expect(router.state.location.href).toBe("/library?view=table");

  await userEvent.click(typeSwitch().getByRole("link", { name: "Anthologies" }));
  await expect.element(page.getByRole("button", { name: "Sort: Publisher" })).toBeVisible();
  expect(router.state.location.href).toBe("/library/anthology?sort=field.publisher");
});

test("a many-Items type's poster shows its status with its tally of counted Items", async () => {
  renderApp({ path: "/library/anthology", handlers: twoTypeLibrary() });
  await expect
    .element(page.getByRole("listitem").filter({ hasText: harborTales.label }))
    .toHaveTextContent(`${harborTales.label}Harbor PressMissing · 12 of 40 met`);
});

test("a type's Library filters by its own listed fields, and All only by the ones every type shares", async () => {
  renderApp({ path: "/library/anthology", handlers: twoTypeLibrary() });
  await userEvent.click(page.getByRole("button", { name: "Filter" }));
  await expect.element(page.getByRole("checkbox", { name: "Harbor Press (1)" })).toBeVisible();
  await userEvent.keyboard("{Escape}");

  await userEvent.click(typeSwitch().getByRole("link", { name: "All" }));
  await expect.element(page.getByRole("link", { name: "Ember Lane" })).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Filter" }));
  await expect.element(page.getByRole("checkbox", { name: "Missing (2)" })).toBeVisible();
  await expect
    .element(page.getByRole("checkbox", { name: "Harbor Press (1)" }))
    .not.toBeInTheDocument();
});
