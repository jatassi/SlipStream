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
  },
);
