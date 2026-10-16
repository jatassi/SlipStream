import { expect, onTestFinished, test, vi } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { ReadOnlyField } from "./read-only-field";
import type { ShellKind } from "./shell-context";

test.each([
  ["desktop", "32px"],
  ["phone", "48px"],
] satisfies [ShellKind, string][])(
  "in the %s shell a read-only value is a %s input that copies its value",
  async (shell, height) => {
    const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue();
    onTestFinished(() => {
      writeText.mockRestore();
    });
    renderKit(<ReadOnlyField label="Log folder" value="/data/home/logs" copy />, { shell });
    const field = page.getByRole("textbox", { name: "Log folder" });
    await expect.element(field).toHaveValue("/data/home/logs");
    await expect.element(field).toHaveAttribute("readonly");
    expect(getComputedStyle(field.element()).height).toBe(height);
    await page.getByRole("button", { name: "Copy" }).click();
    expect(writeText.mock.calls).toEqual([["/data/home/logs"]]);
  },
);

test("a read-only value offers no Copy unless asked", async () => {
  renderKit(<ReadOnlyField label="API key" value="ss_key" description="Shown once." />);
  await expect.element(page.getByText("Shown once.")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Copy" })).not.toBeInTheDocument();
});
