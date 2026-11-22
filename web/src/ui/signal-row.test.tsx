import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type { ShellKind } from "./shell-context";
import { SignalRow } from "./signal-row";

const shells = ["desktop", "phone"] as const satisfies readonly ShellKind[];

const local = (hour: number, minute: number) => new Date(2026, 10, 19, hour, minute).toISOString();

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"], now: new Date(2026, 10, 19, 12, 0) });
});
afterEach(() => {
  vi.useRealTimers();
});

describe.each(shells)("in the %s shell", (shell) => {
  test("a signal row toggles with Enter and Space", async () => {
    renderKit(
      <SignalRow
        signal="fail"
        chip="Failed at Conversion"
        subject="Arrival"
        verdict="The encoder stopped on every Node."
        at={local(9, 12)}
      >
        <p>Attempt 1 on forge</p>
      </SignalRow>,
      { shell },
    );
    const row = page.getByRole("button", {
      name: "Failed at Conversion Arrival The encoder stopped on every Node. 9:12 AM",
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

test("a word before the time reads with it", async () => {
  renderKit(
    <SignalRow
      signal="warn"
      chip="Warning"
      subject="D:"
      verdict="Free up space."
      timeLead="since"
      at={local(14, 5)}
    >
      <p>Free 18 GB</p>
    </SignalRow>,
  );
  await expect
    .element(page.getByRole("button", { name: "Warning D: Free up space. since 2:05 PM" }))
    .toBeVisible();
});
