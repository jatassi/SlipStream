import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import type { ShellKind } from "./shell-context";
import { Spinner } from "./spinner";

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a labelled spinner is a 14px status turning every 0.8s",
  async (shell) => {
    renderKit(<Spinner label="Searching" />, { shell });
    const spinner = page.getByRole("status", { name: "Searching" });
    await expect.element(spinner).toBeVisible();
    const style = getComputedStyle(spinner.element());
    expect([style.width, style.height, style.animationDuration]).toEqual(["14px", "14px", "0.8s"]);
  },
);

test("an unlabelled spinner is hidden from assistive technology", async () => {
  renderKit(
    <button type="button">
      <Spinner />
      Search
    </button>,
  );
  await expect.element(page.getByRole("button", { name: "Search" })).toBeVisible();
  await expect.element(page.getByRole("status")).not.toBeInTheDocument();
});
