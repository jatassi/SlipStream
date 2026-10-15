import { Download04Icon } from "@hugeicons/core-free-icons";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Button } from "./button";

const height = (name: string) =>
  getComputedStyle(page.getByRole("button", { name }).element()).height;

test("a Console button is 32px, a small one 28px and a primary one 32px", async () => {
  renderKit(
    <>
      <Button>Grab</Button>
      <Button size="small">Skip</Button>
      <Button look="primary">Add</Button>
    </>,
    { shell: "desktop" },
  );
  await expect.element(page.getByRole("button", { name: "Grab" })).toBeVisible();
  expect([height("Grab"), height("Skip"), height("Add")]).toEqual(["32px", "28px", "32px"]);
});

test("a Touch button is 44px, a small one keeps 44px and the primary is 50px", async () => {
  renderKit(
    <>
      <Button>Grab</Button>
      <Button size="small">Skip</Button>
      <Button look="primary">Add</Button>
    </>,
    { shell: "phone" },
  );
  await expect.element(page.getByRole("button", { name: "Grab" })).toBeVisible();
  expect([height("Grab"), height("Skip"), height("Add")]).toEqual(["44px", "44px", "50px"]);
});

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
  expect(getComputedStyle(button.element()).opacity).toBe("1");
});

test("a disabled button is 40% opaque and cannot be pressed", async () => {
  renderKit(<Button disabled>Remove</Button>);
  const button = page.getByRole("button", { name: "Remove" });
  await expect.element(button).toBeDisabled();
  expect(getComputedStyle(button.element()).opacity).toBe("0.4");
});
