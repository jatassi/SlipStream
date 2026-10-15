import { describe, expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type { ShellKind } from "./shell-context";
import { StatusSentence } from "./status-sentence";

const shells = [
  { shell: "desktop", size: "24px" },
  { shell: "phone", size: "17px" },
] as const satisfies readonly { shell: ShellKind; size: string }[];

describe.each(shells)("in the $shell shell", ({ shell, size }) => {
  test("the status sentence reads as one sentence with its links", async () => {
    renderKit(
      <StatusSentence
        parts={[
          { kind: "live", text: "5 jobs moving" },
          { kind: "failure", text: "1 failure", href: "#failures" },
          { kind: "warning", text: "relay-fra degraded", href: "#warnings" },
        ]}
      />,
      { shell },
    );
    const line = page.getByRole("paragraph");
    await expect.element(line).toHaveTextContent("5 jobs moving · 1 failure · relay-fra degraded");
    expect(getComputedStyle(line.element()).fontSize).toBe(size);
    await expect
      .element(page.getByRole("link", { name: "1 failure" }))
      .toHaveAttribute("href", "#failures");
    await expect
      .element(page.getByRole("link", { name: "relay-fra degraded" }))
      .toHaveAttribute("href", "#warnings");
  });
});
