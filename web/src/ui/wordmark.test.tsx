import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type { ShellKind } from "./shell-context";
import { Wordmark } from "./wordmark";

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell the wordmark reads SlipStream with Slip at 600 and Stream at 400",
  async (shell) => {
    renderKit(<Wordmark className="text-[30px]" />, { shell });
    await expect.element(page.getByText("SlipStream")).toBeVisible();
    const weights = ["Slip", "Stream"].map(
      (part) => getComputedStyle(page.getByText(part, { exact: true }).element()).fontWeight,
    );
    expect(weights).toEqual(["600", "400"]);
    await expect.element(page.getByRole("img")).not.toBeInTheDocument();
  },
);

test("the mark's box is 1.4em, twice Geist's cap height, a third of it apart from the name", async () => {
  renderKit(<Wordmark className="text-[30px]" />);
  await expect.element(page.getByText("SlipStream")).toBeVisible();
  const mark = document.querySelector("svg");
  const lockup = mark?.parentElement;
  expect([
    mark === null ? null : getComputedStyle(mark).width,
    lockup === null || lockup === undefined ? null : getComputedStyle(lockup).columnGap,
  ]).toEqual(["42px", "14px"]);
});
