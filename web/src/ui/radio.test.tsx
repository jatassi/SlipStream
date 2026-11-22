import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { RadioGroup } from "./radio";

const orders = [
  { value: "added", label: "Date added" },
  { value: "title", label: "Title" },
  { value: "year", label: "Year" },
] as const;

function Order() {
  const [value, setValue] = useState<(typeof orders)[number]["value"]>("added");
  return <RadioGroup label="Sort by" options={orders} value={value} onValueChange={setValue} />;
}

test("a radio group's rows select with arrows and clicks", async () => {
  renderKit(<Order />);
  await expect.element(page.getByRole("radiogroup", { name: "Sort by" })).toBeVisible();
  await expect.element(page.getByRole("radio", { name: "Date added" })).toBeChecked();
  await userEvent.tab();
  await userEvent.keyboard("{ArrowDown}");
  await expect.element(page.getByRole("radio", { name: "Title" })).toBeChecked();
  await page.getByText("Year").click();
  await expect.element(page.getByRole("radio", { name: "Year" })).toBeChecked();
});
