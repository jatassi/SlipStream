import { describe, expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Section } from "./section";
import type { ShellKind } from "./shell-context";

const shells = [
  { shell: "desktop", gap: "24px", title: "14px" },
  { shell: "phone", gap: "28px", title: "20px" },
] as const satisfies readonly { shell: ShellKind; gap: string; title: string }[];

describe.each(shells)("in the $shell shell", ({ shell, gap, title }) => {
  test("sections are named regions spaced apart", async () => {
    renderKit(
      <>
        <Section title="Needs you" count={2}>
          <p>Two failures</p>
        </Section>
        <Section title="Moving">
          <p>Five jobs</p>
        </Section>
      </>,
      { shell },
    );
    const needs = page.getByRole("region", { name: "Needs you" });
    const moving = page.getByRole("region", { name: "Moving" });
    await expect.element(needs.getByRole("heading", { level: 2 })).toHaveTextContent("Needs you2");
    await expect.element(moving.getByText("Five jobs")).toBeVisible();
    expect(getComputedStyle(needs.element()).marginTop).toBe("0px");
    expect(getComputedStyle(moving.element()).marginTop).toBe(gap);
    expect(getComputedStyle(moving.getByRole("heading").element()).fontSize).toBe(title);
  });
});
