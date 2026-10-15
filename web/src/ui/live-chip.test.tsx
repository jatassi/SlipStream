import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { LiveChip } from "./live-chip";
import type { ShellKind } from "./shell-context";

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a live chip is a 24px chip whose dot glows",
  async (shell) => {
    renderKit(<LiveChip>Downloading</LiveChip>, { shell });
    const chip = page.getByText("Downloading");
    await expect.element(chip).toBeVisible();
    expect(getComputedStyle(chip.element()).height).toBe("24px");
    const dot = chip.element().querySelector("[aria-hidden]");
    expect(dot === null ? "none" : getComputedStyle(dot).boxShadow).toContain("10px");
  },
);
