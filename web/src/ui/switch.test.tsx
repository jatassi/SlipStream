import { useState } from "react";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";

import { Switch } from "./switch";

function Startup() {
  const [on, setOn] = useState(true);
  return <Switch label="Play the startup sequence" checked={on} onCheckedChange={setOn} />;
}

const name = "Play the startup sequence";

test("Space toggles a switch and its label is the target", async () => {
  renderKit(<Startup />);
  const control = page.getByRole("switch", { name });
  await expect.element(control).toBeChecked();
  await userEvent.tab();
  await expect.element(control).toHaveFocus();
  await userEvent.keyboard(" ");
  await expect.element(control).not.toBeChecked();
  await page.getByText(name).click();
  await expect.element(control).toBeChecked();
});

function Season() {
  const [on, setOn] = useState<boolean | "mixed">("mixed");
  return <Switch label="Monitor season 2" checked={on} onCheckedChange={setOn} />;
}

test("a mixed switch reads mixed and turns on", async () => {
  renderKit(<Season />);
  const control = page.getByRole("switch", { name: "Monitor season 2" });
  await expect.element(control).toHaveAttribute("aria-checked", "mixed");
  await control.click();
  await expect.element(control).toHaveAttribute("aria-checked", "true");
  await control.click();
  await expect.element(control).toHaveAttribute("aria-checked", "false");
});
