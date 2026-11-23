import { Download04Icon } from "@hugeicons/core-free-icons";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Button } from "./button";

test("a click runs the action and Enter presses it from the keyboard", async () => {
  let pressed = 0;
  renderKit(
    <Button
      icon={Download04Icon}
      onClick={() => {
        pressed += 1;
      }}
    >
      Search
    </Button>,
  );
  await page.getByRole("button", { name: "Search" }).click();
  await userEvent.keyboard("{Enter}");
  expect(pressed).toBe(2);
});

test("a busy button keeps its label and focus and ignores presses", async () => {
  let pressed = 0;
  renderKit(
    <Button
      busy
      icon={Download04Icon}
      onClick={() => {
        pressed += 1;
      }}
    >
      Search
    </Button>,
    { shell: "phone" },
  );
  const button = page.getByRole("button", { name: "Search" });
  await expect.element(button).toHaveAttribute("aria-busy", "true");
  await userEvent.tab();
  await expect.element(button).toHaveFocus();
  await userEvent.keyboard("{Enter}");
  expect(pressed).toBe(0);
});
