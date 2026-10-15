import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Count } from "./count";
import type { ShellKind } from "./shell-context";

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a count reads its number and the word its dot stands for",
  async (shell) => {
    renderKit(
      <ul>
        <li>
          Activity <Count value={2} dot="live" />
        </li>
        <li>
          Health <Count value={1} dot="warn" />
        </li>
        <li>
          Imports <Count value={3} dot="fail" />
        </li>
        <li>
          Library <Count value={240} />
        </li>
      </ul>,
      { shell },
    );
    const rows = page.getByRole("listitem");
    await expect.element(rows.nth(0)).toHaveTextContent("Activity 2 live");
    await expect.element(rows.nth(1)).toHaveTextContent("Health 1 needs attention");
    await expect.element(rows.nth(2)).toHaveTextContent("Imports 3 failed");
    await expect.element(rows.nth(3)).toHaveTextContent("Library 240");
    const style = getComputedStyle(page.getByText("240").element());
    expect([style.fontSize, style.fontWeight, style.fontVariantNumeric]).toEqual([
      "11px",
      "500",
      "tabular-nums",
    ]);
  },
);
