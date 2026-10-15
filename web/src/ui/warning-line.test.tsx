import { describe, expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type { ShellKind } from "./shell-context";
import { WarningLine } from "./warning-line";

const shells = ["desktop", "phone"] as const satisfies readonly ShellKind[];

describe.each(shells)("in the %s shell", (shell) => {
  test("a warning line names its subject and offers one warn action", async () => {
    const fixed: string[] = [];
    renderKit(
      <WarningLine
        chip="Degraded"
        subject="relay-fra"
        detail="Half its probes time out."
        since="08:30"
        action={{
          label: "Check relay",
          onClick: () => {
            fixed.push("check");
          },
        }}
      />,
      { shell },
    );
    await expect.element(page.getByText("Degraded")).toBeVisible();
    await expect.element(page.getByText("since 08:30")).toBeVisible();
    await page.getByRole("button", { name: "Check relay" }).click();
    expect(fixed).toStrictEqual(["check"]);
  });
});
