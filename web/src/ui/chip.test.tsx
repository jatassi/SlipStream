import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Chip } from "./chip";
import type { ShellKind } from "./shell-context";

test.each(["desktop", "phone"] satisfies ShellKind[])(
  "in the %s shell a chip is 24px of 12px text",
  async (shell) => {
    renderKit(<Chip>1080p</Chip>, { shell });
    const chip = page.getByText("1080p");
    await expect.element(chip).toBeVisible();
    const style = getComputedStyle(chip.element());
    expect([style.height, style.fontSize, style.borderRadius]).toEqual(["24px", "12px", "6px"]);
  },
);

test("an outlined chip has a hairline ring and no fill", async () => {
  renderKit(
    <>
      <Chip>1080p</Chip>
      <Chip look="outline">Downloading</Chip>
    </>,
  );
  await expect.element(page.getByText("Downloading")).toBeVisible();
  const filled = getComputedStyle(page.getByText("1080p").element());
  const outlined = getComputedStyle(page.getByText("Downloading").element());
  expect(outlined.backgroundColor).toBe("rgba(0, 0, 0, 0)");
  expect(filled.backgroundColor).not.toBe("rgba(0, 0, 0, 0)");
  expect(outlined.boxShadow).toMatch(/ 0px 0px 0px 1px inset/u);
  expect([outlined.height, outlined.fontSize]).toEqual(["24px", "12px"]);
});
