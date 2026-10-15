import { describe, expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Inspector, InspectorSection } from "./inspector";
import type { ShellKind } from "./shell-context";

const shells = ["desktop", "phone"] as const satisfies readonly ShellKind[];

describe.each(shells)("in the %s shell", (shell) => {
  test("the Inspector is a labelled complementary landmark that closes", async () => {
    const closed: string[] = [];
    renderKit(
      <Inspector
        title="Dune: Part Two"
        onClose={() => {
          closed.push("close");
        }}
      >
        <InspectorSection title="Release">
          <p>Dune.Part.Two.2024.2160p</p>
        </InspectorSection>
      </Inspector>,
      { shell },
    );
    const panel = page.getByRole("complementary", { name: "Dune: Part Two" });
    await expect.element(panel.getByRole("region", { name: "Release" })).not.toBeInTheDocument();
    await expect.element(panel.getByRole("heading", { name: "Release" })).toBeVisible();
    await panel.getByRole("button", { name: "Close" }).click();
    expect(closed).toStrictEqual(["close"]);
  });
});

test("the Inspector is min(480px, 40vw) wide", async () => {
  renderKit(
    <Inspector title="Arrival" onClose={() => {}}>
      <p>Detail</p>
    </Inspector>,
  );
  const panel = page.getByRole("complementary", { name: "Arrival" });
  await expect.element(panel).toBeVisible();
  expect(getComputedStyle(panel.element()).width).toBe("480px");
});
