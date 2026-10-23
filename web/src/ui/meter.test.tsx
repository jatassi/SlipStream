import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Meter } from "./meter";
import type { ShellKind } from "./shell-context";

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a meter is a named meter with its value",
  async (shell) => {
    renderKit(<Meter label="Storage used" value={600} max={1000} />, { shell });
    const meter = page.getByRole("meter", { name: "Storage used" });
    await expect.element(meter).toHaveAttribute("aria-valuenow", "600");
    await expect.element(meter).toHaveAttribute("aria-valuemax", "1000");
    await expect.element(meter).toHaveAttribute("aria-valuetext", "60%");
    expect(meter.element().textContent).toBe("");
  },
);

const fillOf = (meter: Element) =>
  getComputedStyle(meter.querySelector("[class*=rounded-capsule] > *") ?? meter).backgroundColor;

test.each(["warn", "fail"] as const)("a %s meter fills in that signal's colour", async (tone) => {
  renderKit(
    <>
      <Meter label="Plain" value={600} max={1000} />
      <Meter label="Toned" value={600} max={1000} tone={tone} />
      <span data-testid="signal" className={`bg-${tone}`} />
    </>,
  );
  const toned = page.getByRole("meter", { name: "Toned" });
  await expect.element(toned).toBeVisible();
  const signal = getComputedStyle(page.getByTestId("signal").element()).backgroundColor;
  expect(fillOf(toned.element())).toBe(signal);
  expect(fillOf(page.getByRole("meter", { name: "Plain" }).element())).not.toBe(signal);
});
