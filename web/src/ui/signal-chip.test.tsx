import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type { ShellKind } from "./shell-context";
import { SignalChip } from "./signal-chip";

test.each([
  ["fail", "Failed at Import", "rgb(255, 107, 97)"],
  ["warn", "Stalled", "rgb(255, 181, 71)"],
  ["ok", "Healthy", "rgb(95, 211, 148)"],
] as const)(
  "a %s signal chip says %s at 700 on solid %s with its icon",
  async (kind, word, fill) => {
    renderKit(<SignalChip kind={kind}>{word}</SignalChip>);
    const chip = page.getByText(word);
    await expect.element(chip).toBeVisible();
    const style = getComputedStyle(chip.element());
    expect([style.fontWeight, style.backgroundColor]).toEqual(["700", fill]);
    expect(chip.element().querySelector("svg")).not.toBeNull();
  },
);

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a signal chip is 24px",
  async (shell) => {
    renderKit(<SignalChip kind="fail">Failed</SignalChip>, { shell });
    await expect.element(page.getByText("Failed")).toBeVisible();
    expect(getComputedStyle(page.getByText("Failed").element()).height).toBe("24px");
  },
);
