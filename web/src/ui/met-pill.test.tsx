import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { MetPill } from "./met-pill";
import type { ShellKind } from "./shell-context";

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell Met is a 28px solid white capsule with a check",
  async (shell) => {
    renderKit(<MetPill>Met</MetPill>, { shell });
    const pill = page.getByText("Met");
    await expect.element(pill).toBeVisible();
    const style = getComputedStyle(pill.element());
    expect([style.height, style.borderRadius, style.backgroundColor]).toEqual([
      "28px",
      "999px",
      "rgb(246, 244, 240)",
    ]);
    expect(pill.element().querySelector("svg")).not.toBeNull();
  },
);
