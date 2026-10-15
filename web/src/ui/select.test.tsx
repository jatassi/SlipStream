import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Select } from "./select";
import type { ShellKind } from "./shell-context";

const profiles = [
  { value: "any", label: "Any" },
  { value: "hd", label: "HD 1080p" },
  { value: "uhd", label: "Ultra HD" },
] as const;

function Profile() {
  const [value, setValue] = useState<(typeof profiles)[number]["value"]>("any");
  return <Select label="Quality" options={profiles} value={value} onValueChange={setValue} />;
}

test.each([
  ["desktop", "32px", "36px"],
  ["phone", "48px", "52px"],
] satisfies [ShellKind, string, string][])(
  "in the %s shell a %s select opens %s options, carrying the density it opened from, and picks one from the keyboard",
  async (shell, trigger, item) => {
    renderKit(<Profile />, { shell });
    const select = page.getByRole("combobox", { name: "Quality" });
    await expect.element(select).toHaveTextContent("Any");
    expect(getComputedStyle(select.element()).height).toBe(trigger);
    await userEvent.tab();
    await userEvent.keyboard("{ArrowDown}");
    const option = page.getByRole("option", { name: "HD 1080p" });
    await expect.element(option).toBeVisible();
    expect(getComputedStyle(option.element()).height).toBe(item);
    await userEvent.keyboard("{ArrowDown}");
    await expect.element(option).toHaveFocus();
    await userEvent.keyboard("{Enter}");
    await expect.element(select).toHaveTextContent("HD 1080p");
    await expect.element(option).not.toBeInTheDocument();
  },
);

test("Esc closes the popup and leaves the value", async () => {
  renderKit(<Profile />);
  await page.getByRole("combobox", { name: "Quality" }).click();
  await expect.element(page.getByRole("listbox")).toBeVisible();
  await userEvent.keyboard("{Escape}");
  await expect.element(page.getByRole("listbox")).not.toBeInTheDocument();
  await expect.element(page.getByRole("combobox", { name: "Quality" })).toHaveTextContent("Any");
});
