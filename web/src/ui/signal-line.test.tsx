import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { SignalLine } from "./signal-line";

test("an announced failure is an alert with its sentence", async () => {
  renderKit(
    <SignalLine tone="fail" announce>
      That PIN is wrong.
    </SignalLine>,
  );
  await expect.element(page.getByRole("alert")).toHaveTextContent("That PIN is wrong.");
});

test.each(["warn", "ok"] as const)("an announced %s line is a status", async (tone) => {
  renderKit(
    <SignalLine tone={tone} announce>
      The Test passed.
    </SignalLine>,
  );
  await expect.element(page.getByRole("status")).toHaveTextContent("The Test passed.");
});

test("a line drawn with the page is neither an alert nor a status", async () => {
  renderKit(<SignalLine tone="fail">TorrentLeech refused the sign-in.</SignalLine>);
  await expect.element(page.getByText("TorrentLeech refused the sign-in.")).toBeVisible();
  await expect.element(page.getByRole("alert")).not.toBeInTheDocument();
  await expect.element(page.getByRole("status")).not.toBeInTheDocument();
});

test("each tone draws its icon in the signal's hue and its words in the tone's ink", async () => {
  renderKit(
    <>
      <SignalLine tone="ok">Passed.</SignalLine>
      <span data-testid="ok" className="text-ok" />
      <span data-testid="ink" className="text-ink-1" />
    </>,
  );
  await expect.element(page.getByText("Passed.")).toBeVisible();
  const line = page.getByText("Passed.").element().parentElement;
  const icon = line?.querySelector("svg");
  expect(icon && getComputedStyle(icon).color).toBe(
    getComputedStyle(page.getByTestId("ok").element()).color,
  );
  expect(line && getComputedStyle(line).color).toBe(
    getComputedStyle(page.getByTestId("ink").element()).color,
  );
});
