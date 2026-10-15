import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Segmented } from "./segmented";
import type { ShellKind } from "./shell-context";

const filters = [
  { value: "all", label: "All", count: 12 },
  { value: "moving", label: "Moving", count: 3 },
  { value: "failed", label: "Failed", count: 1 },
] as const;

function Filter() {
  const [value, setValue] = useState<(typeof filters)[number]["value"]>("all");
  return <Segmented label="Show" options={filters} value={value} onValueChange={setValue} />;
}

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a segmented control is a radiogroup that arrows move through",
  async (shell) => {
    renderKit(<Filter />, { shell });
    await expect.element(page.getByRole("radiogroup", { name: "Show" })).toBeVisible();
    await expect.element(page.getByRole("radio", { name: "All 12" })).toBeChecked();
    await userEvent.tab();
    await expect.element(page.getByRole("radio", { name: "All 12" })).toHaveFocus();
    await userEvent.keyboard("{ArrowRight}");
    await expect.element(page.getByRole("radio", { name: "Moving 3" })).toBeChecked();
    await page.getByRole("radio", { name: "Failed 1" }).click();
    await expect.element(page.getByRole("radio", { name: "Failed 1" })).toBeChecked();
    await expect.element(page.getByRole("radio", { name: "All 12" })).not.toBeChecked();
  },
);

test.each([
  ["desktop", "28px"],
  ["phone", "40px"],
] satisfies [ShellKind, string][])(
  "in the %s shell an option is %s tall",
  async (shell, height) => {
    renderKit(<Filter />, { shell });
    const option = page.getByRole("radio", { name: "Moving 3" });
    await expect.element(option).toBeVisible();
    expect(getComputedStyle(option.element()).height).toBe(height);
  },
);

test("on phone the control spans its container", async () => {
  renderKit(
    <div style={{ width: "360px" }}>
      <Filter />
    </div>,
    { shell: "phone" },
  );
  const group = page.getByRole("radiogroup", { name: "Show" });
  await expect.element(group).toBeVisible();
  expect(getComputedStyle(group.element()).width).toBe("360px");
});
