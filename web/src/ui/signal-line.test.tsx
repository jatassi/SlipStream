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
