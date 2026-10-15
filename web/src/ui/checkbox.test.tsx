import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Checkbox } from "./checkbox";
import type { ShellKind } from "./shell-context";

function Monitor() {
  const [on, setOn] = useState(false);
  return <Checkbox label="Monitor" checked={on} onCheckedChange={setOn} />;
}

test.each([
  ["desktop", "16px"],
  ["phone", "22px"],
] satisfies [ShellKind, string][])(
  "in the %s shell a %s checkbox toggles with Space and with a click on its label",
  async (shell, size) => {
    renderKit(<Monitor />, { shell });
    const box = page.getByRole("checkbox", { name: "Monitor" });
    await expect.element(box).not.toBeChecked();
    expect(getComputedStyle(box.element()).width).toBe(size);
    await userEvent.tab();
    await userEvent.keyboard(" ");
    await expect.element(box).toBeChecked();
    await page.getByText("Monitor").click();
    await expect.element(box).not.toBeChecked();
  },
);
