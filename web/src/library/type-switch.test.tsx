import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { cinemaLibrary, harborTales, twoTypeLibrary } from "../test/library-page";
import { renderApp } from "../test/render-app";

const typeSwitch = () => page.getByRole("navigation", { name: "Content Types" });

test.each(["/library/cinema"])("with one type on, %s has no type switch", async (path) => {
  renderApp({ path, handlers: cinemaLibrary().handlers });
  await expect.element(page.getByRole("link", { name: "Ember Lane" })).toBeVisible();
  await expect.element(typeSwitch()).not.toBeInTheDocument();
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
    .toHaveTextContent(
      `${harborTales.label}${harborTales.label}Harbor PressMissing · 12 of 40 stories met`,
    );
});
