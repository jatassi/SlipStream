import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { GuidedSteps } from "./guided-steps";

const steps = [
  {
    label: "Open the key page.",
    link: { label: "Open SubDL", href: "https://subdl.invalid/panel" },
  },
  { label: "Generate a key and copy it." },
  { label: "Paste the key here.", children: <input aria-label="Key" /> },
];

const list = () => page.getByRole("list", { name: "Steps to add a key" });

const stepTexts = () =>
  list()
    .getByRole("listitem")
    .all()
    .map((item) => item.element().querySelector("p")?.textContent);

test("steps are numbered in order and the first is current", async () => {
  renderKit(<GuidedSteps label="Steps to add a key" steps={steps} />);
  await expect.element(list()).toBeVisible();
  expect(stepTexts()).toEqual([
    "Step 1: Open the key page.",
    "Step 2: Generate a key and copy it.",
    "Step 3: Paste the key here.",
  ]);
  await expect.element(list().getByRole("listitem").nth(0)).toHaveAttribute("aria-current", "step");
  await expect.element(page.getByRole("textbox", { name: "Key" })).toBeVisible();
});

test("following a step's link opens the real page in a new tab and ticks the step", async () => {
  renderKit(<GuidedSteps label="Steps to add a key" steps={steps} />);
  const link = page.getByRole("link", { name: "Open SubDL" });
  await expect.element(link).toHaveAttribute("href", "https://subdl.invalid/panel");
  await expect.element(link).toHaveAttribute("target", "_blank");
  link.element().addEventListener("click", (event) => {
    event.preventDefault();
  });
  await userEvent.click(link);
  await expect
    .poll(stepTexts)
    .toEqual([
      "Step 1, done: Open the key page.",
      "Step 2: Generate a key and copy it.",
      "Step 3: Paste the key here.",
    ]);
  await expect.element(list().getByRole("listitem").nth(1)).toHaveAttribute("aria-current", "step");
});

test("a step marked done is ticked, and the next is current", async () => {
  renderKit(
    <GuidedSteps
      label="Steps to add a key"
      steps={[{ label: "Open the key page.", done: true }, ...steps.slice(1)]}
    />,
  );
  await expect.element(list().getByRole("listitem").nth(1)).toHaveAttribute("aria-current", "step");
  expect(stepTexts()[0]).toBe("Step 1, done: Open the key page.");
});
