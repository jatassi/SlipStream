import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Chip } from "./chip";
import type { ShellKind } from "./shell-context";

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a chip is 24px of 12px text",
  async (shell) => {
    renderKit(<Chip>1080p</Chip>, { shell });
    const chip = page.getByText("1080p");
    await expect.element(chip).toBeVisible();
    const style = getComputedStyle(chip.element());
    expect([style.height, style.fontSize, style.borderRadius]).toEqual(["24px", "12px", "6px"]);
  },
);
