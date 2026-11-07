import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type { ShellKind } from "./shell-context";
import { Switch } from "./switch";

function Startup() {
  const [on, setOn] = useState(true);
  return <Switch label="Play the startup sequence" checked={on} onCheckedChange={setOn} />;
}

const name = "Play the startup sequence";

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell Space toggles a switch and its label is the target",
  async (shell) => {
    renderKit(<Startup />, { shell });
    const control = page.getByRole("switch", { name });
    await expect.element(control).toBeChecked();
    await userEvent.tab();
    await expect.element(control).toHaveFocus();
    await userEvent.keyboard(" ");
    await expect.element(control).not.toBeChecked();
    await page.getByText(name).click();
    await expect.element(control).toBeChecked();
  },
);

test.each([
  ["desktop", ["34px", "20px"]],
  ["phone", ["51px", "31px"]],
] satisfies [ShellKind, string[]][])("in the %s shell the switch is %j", async (shell, size) => {
  renderKit(<Startup />, { shell });
  const control = page.getByRole("switch", { name });
  await expect.element(control).toBeVisible();
  const style = getComputedStyle(control.element());
  expect([style.width, style.height]).toEqual(size);
});

function knobOffset(control: Element) {
  const track = control.getBoundingClientRect();
  const knob = control.firstElementChild?.getBoundingClientRect();
  return knob === undefined ? NaN : knob.left + knob.width / 2 - (track.left + track.width / 2);
}

function Season() {
  const [on, setOn] = useState<boolean | "mixed">("mixed");
  return <Switch label="Monitor season 2" checked={on} onCheckedChange={setOn} />;
}

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a mixed switch reads mixed with its knob centred, and turns on",
  async (shell) => {
    renderKit(<Season />, { shell });
    const control = page.getByRole("switch", { name: "Monitor season 2" });
    await expect.element(control).toHaveAttribute("aria-checked", "mixed");
    await expect.poll(() => Math.abs(knobOffset(control.element()))).toBeLessThan(0.5);
    await control.click();
    await expect.element(control).toHaveAttribute("aria-checked", "true");
    await control.click();
    await expect.element(control).toHaveAttribute("aria-checked", "false");
  },
);
