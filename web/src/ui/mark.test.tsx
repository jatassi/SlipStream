import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Mark } from "./mark";
import type { ShellKind } from "./shell-context";

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell the mark is an image named SlipStream at its size",
  async (shell) => {
    renderKit(<Mark size={16} viewTransitionName="brand" />, { shell });
    const mark = page.getByRole("img", { name: "SlipStream" });
    await expect.element(mark).toBeVisible();
    const style = getComputedStyle(mark.element());
    expect([style.width, style.height, style.viewTransitionName]).toEqual([
      "16px",
      "16px",
      "brand",
    ]);
  },
);

test("the S takes ink-1 and the dots take the glow", async () => {
  renderKit(<Mark />);
  const mark = page.getByRole("img", { name: "SlipStream" });
  await expect.element(mark).toBeVisible();
  const [s, dot] = [mark.element().querySelector("path"), mark.element().querySelector("circle")];
  expect([
    s === null ? null : getComputedStyle(s).stroke,
    dot === null ? null : getComputedStyle(dot).fill,
  ]).toEqual(["rgb(242, 244, 247)", "rgb(116, 175, 251)"]);
});

test("a decorative mark is hidden from assistive technology", async () => {
  renderKit(
    <a href="/">
      <Mark decorative />
      Home
    </a>,
  );
  await expect.element(page.getByRole("link", { name: "Home" })).toBeVisible();
  await expect.element(page.getByRole("img")).not.toBeInTheDocument();
});
