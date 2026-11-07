import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { harborTales, twoTypeLibrary } from "../test/library-page";
import { renderApp } from "../test/render-app";

const headers = (table: string) =>
  page
    .getByRole("table", { name: table })
    .getByRole("columnheader")
    .elements()
    .map((header) => header.textContent);

test("a many-Items type's table names its availability column and sort its own way and offers its listed fields", async () => {
  renderApp({
    path: "/library/anthology?view=table&cols=status%2Cavailable%2Cfield.publisher&sort=available",
    handlers: twoTypeLibrary(),
  });
  const table = page.getByRole("table", { name: "Anthologies" });
  await expect.element(table).toBeVisible();
  expect(headers("Anthologies")).toEqual(
    expect.arrayContaining(["Status", "Next issue", "Publisher"]),
  );
  await expect
    .element(table.getByRole("row").filter({ hasText: harborTales.label }))
    .toHaveTextContent(
      `${harborTales.label}Harbor PressMissing · 12 of 40 metDec 1, 2026Harbor Press`,
    );
  await expect.element(page.getByRole("button", { name: "Sort: Next issue" })).toBeVisible();
});

test("All labels the availability column Available and offers no field only one type lists", async () => {
  renderApp({ path: "/library?view=table&cols=status%2Cavailable", handlers: twoTypeLibrary() });
  await expect.element(page.getByRole("table", { name: "Works" })).toBeVisible();
  expect(headers("Works")).toEqual(expect.arrayContaining(["Status", "Available"]));
  expect(headers("Works")).not.toContain("Next issue");

  await userEvent.click(page.getByRole("button", { name: "Library options" }));
  await expect.element(page.getByRole("menuitemcheckbox", { name: "Available" })).toBeVisible();
  await expect
    .element(page.getByRole("menuitemcheckbox", { name: "Publisher" }))
    .not.toBeInTheDocument();
  await expect
    .element(page.getByRole("menuitemcheckbox", { name: "Year" }))
    .not.toBeInTheDocument();
});
