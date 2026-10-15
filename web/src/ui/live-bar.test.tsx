import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { LiveBar } from "./live-bar";
import type { ShellKind } from "./shell-context";

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a live bar is a named progressbar on a 6px capsule track",
  async (shell) => {
    renderKit(<LiveBar label="Downloading Dune" value={42} />, { shell });
    const bar = page.getByRole("progressbar", { name: "Downloading Dune" });
    await expect.element(bar).toHaveAttribute("aria-valuenow", "42");
    const track = bar.element().firstElementChild;
    expect(track === null ? null : getComputedStyle(track).height).toBe("6px");
  },
);
