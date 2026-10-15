import { Cancel01Icon } from "@hugeicons/core-free-icons";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type { ShellKind } from "./shell-context";
import { IconButton } from "./icon-button";

test.each([
  ["desktop", "32px"],
  ["phone", "44px"],
] satisfies [ShellKind, string][])(
  "in the %s shell an icon button is a %s circle",
  async (shell, size) => {
    renderKit(<IconButton icon={Cancel01Icon} label="Close" />, { shell });
    const button = page.getByRole("button", { name: "Close" });
    await expect.element(button).toBeVisible();
    const style = getComputedStyle(button.element());
    expect([style.width, style.height, style.borderRadius]).toEqual([size, size, "999px"]);
  },
);

test("Space presses an icon button named by its label", async () => {
  let pressed = 0;
  renderKit(
    <IconButton
      icon={Cancel01Icon}
      label="Close"
      onClick={() => {
        pressed += 1;
      }}
    />,
  );
  await userEvent.tab();
  await expect.element(page.getByRole("button", { name: "Close" })).toHaveFocus();
  await userEvent.keyboard(" ");
  expect(pressed).toBe(1);
});
