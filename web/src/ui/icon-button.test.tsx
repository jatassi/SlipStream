import { Cancel01Icon } from "@hugeicons/core-free-icons";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { IconButton } from "./icon-button";

test("Space presses an icon button named by its label", async () => {
  let pressed = 0;
  renderKit(
    <IconButton
      icon={Cancel01Icon}
      label="Close"
      onClick={() => {
        pressed += 1;
      }}
    />,
  );
  const button = page.getByRole("button", { name: "Close" });
  await expect.element(button).toBeVisible();
  await userEvent.tab();
  await expect.element(button).toHaveFocus();
  await userEvent.keyboard(" ");
  expect(pressed).toBe(1);
});

test("a busy icon button shows a spinner in its icon's place, keeps its focus and can't be pressed", async () => {
  let pressed = 0;
  renderKit(
    <IconButton
      icon={Cancel01Icon}
      label="Search Volume 3"
      busy
      onClick={() => {
        pressed += 1;
      }}
    />,
  );
  const button = page.getByRole("button", { name: "Search Volume 3" });
  await expect.element(button).toHaveAttribute("aria-busy", "true");
  await userEvent.tab();
  await expect.element(button).toHaveFocus();
  await userEvent.keyboard(" ");
  expect(pressed).toBe(0);
});
