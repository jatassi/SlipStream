import { describe, expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type { ShellKind } from "./shell-context";
import { SignalRow } from "./signal-row";

const shells = ["desktop", "phone"] as const satisfies readonly ShellKind[];

describe.each(shells)("in the %s shell", (shell) => {
  test("a signal row toggles with Enter and Space", async () => {
    renderKit(
      <SignalRow
        signal="fail"
        chip="Failed at Conversion"
        subject="Arrival"
        verdict="The encoder stopped on every Node."
        time="09:12"
      >
        <p>Attempt 1 on forge</p>
      </SignalRow>,
      { shell },
    );
    const row = page.getByRole("button", {
      name: "Failed at Conversion Arrival The encoder stopped on every Node. 09:12",
    });
    await expect.element(row).toHaveAttribute("aria-expanded", "false");
    await expect.element(page.getByText("Attempt 1 on forge")).not.toBeInTheDocument();
    row.element().focus();
    await userEvent.keyboard("{Enter}");
    await expect.element(row).toHaveAttribute("aria-expanded", "true");
    await expect.element(page.getByText("Attempt 1 on forge")).toBeVisible();
    await userEvent.keyboard(" ");
    await expect.element(row).toHaveAttribute("aria-expanded", "false");
    await expect.element(page.getByText("Attempt 1 on forge")).not.toBeInTheDocument();
  });
});
