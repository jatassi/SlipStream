import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Segmented } from "./segmented";

const filters = [
  { value: "all", label: "All", count: 12 },
  { value: "moving", label: "Moving", count: 3 },
  { value: "failed", label: "Failed", count: 1 },
] as const;

function Filter() {
  const [value, setValue] = useState<(typeof filters)[number]["value"]>("all");
  return <Segmented label="Show" options={filters} value={value} onValueChange={setValue} />;
}

test("a segmented control is a radiogroup that arrows move through", async () => {
  renderKit(<Filter />);
  await expect.element(page.getByRole("radiogroup", { name: "Show" })).toBeVisible();
  await expect.element(page.getByRole("radio", { name: "All 12" })).toBeChecked();
  await userEvent.tab();
  await expect.element(page.getByRole("radio", { name: "All 12" })).toHaveFocus();
  await userEvent.keyboard("{ArrowRight}");
  await expect.element(page.getByRole("radio", { name: "Moving 3" })).toBeChecked();
  await page.getByRole("radio", { name: "Failed 1" }).click();
  await expect.element(page.getByRole("radio", { name: "Failed 1" })).toBeChecked();
  await expect.element(page.getByRole("radio", { name: "All 12" })).not.toBeChecked();
});
