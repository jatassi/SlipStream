import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { RadioGroup } from "./radio";
import type { ShellKind } from "./shell-context";

const orders = [
  { value: "added", label: "Date added" },
  { value: "title", label: "Title" },
  { value: "year", label: "Year" },
] as const;

function Order() {
  const [value, setValue] = useState<(typeof orders)[number]["value"]>("added");
  return <RadioGroup label="Sort by" options={orders} value={value} onValueChange={setValue} />;
}

test.each([
  ["desktop", "32px"],
  ["phone", "44px"],
] satisfies [ShellKind, string][])(
  "in the %s shell a radio group's %s rows select with arrows and clicks",
  async (shell, row) => {
    renderKit(<Order />, { shell });
    await expect.element(page.getByRole("radiogroup", { name: "Sort by" })).toBeVisible();
    await expect.element(page.getByRole("radio", { name: "Date added" })).toBeChecked();
    expect(getComputedStyle(page.getByText("Title").element()).height).toBe(row);
    await userEvent.tab();
    await userEvent.keyboard("{ArrowDown}");
    await expect.element(page.getByRole("radio", { name: "Title" })).toBeChecked();
    await page.getByText("Year").click();
    await expect.element(page.getByRole("radio", { name: "Year" })).toBeChecked();
  },
);
