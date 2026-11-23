import { HourglassIcon } from "@hugeicons/core-free-icons";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Mark } from "./state-mark";
import { type StatusMark, statusMarks } from "./state-marks";

const stalled: StatusMark = {
  word: "Stalled",
  icon: HourglassIcon,
  tone: "neutral",
  definition: "The torrent is incomplete, and no peer is sending it data.",
};

const definition = () => page.getByRole("dialog", { name: "Stalled" });

function tap(element: Element) {
  const touch = { bubbles: true, cancelable: true, pointerType: "touch", isPrimary: true };
  element.dispatchEvent(new PointerEvent("pointerdown", touch));
  element.dispatchEvent(new PointerEvent("pointerup", touch));
  element.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true }));
}

test("a defined mark is a button named by its word and described by its definition", async () => {
  renderKit(<Mark mark={stalled} />);
  const trigger = page.getByRole("button", { name: "Stalled" });
  await expect.element(trigger).toHaveAccessibleName("Stalled");
  await expect
    .element(trigger)
    .toHaveAccessibleDescription("The torrent is incomplete, and no peer is sending it data.");
});

test("hovering a defined mark opens its definition", async () => {
  renderKit(<Mark mark={stalled} />);
  await userEvent.hover(page.getByRole("button", { name: "Stalled" }));
  await expect.element(definition()).toMatchTextContent(/incomplete.+no peer/u);
});

test("a tap opens the definition in the touch density", async () => {
  renderKit(<Mark mark={stalled} />, { density: "touch" });
  const trigger = page.getByRole("button", { name: "Stalled" });
  await expect.element(trigger).toBeVisible();
  tap(trigger.element());
  await expect.element(definition()).toBeVisible();
  expect(definition().element().closest<HTMLElement>("[data-density]")?.dataset["density"]).toBe(
    "touch",
  );
});

test("Enter opens the definition and keeps focus on the mark, and Esc closes it", async () => {
  renderKit(<Mark mark={stalled} />);
  const trigger = page.getByRole("button", { name: "Stalled" });
  await userEvent.keyboard("{Tab}");
  await expect.element(trigger).toHaveFocus();
  await userEvent.keyboard("{Enter}");
  await expect.element(definition()).toBeVisible();
  await expect.element(trigger).toHaveFocus();

  await userEvent.keyboard("{Escape}");
  await expect.element(definition()).not.toBeInTheDocument();
  await expect.element(trigger).toHaveFocus();
});

test("a mark without a definition draws its icon and word with no button", async () => {
  renderKit(<Mark mark={statusMarks.missing} tally="8 of 10 met" />);
  const missing = page.getByText("Missing · 8 of 10 met");
  await expect.element(missing).toBeVisible();
  await expect.element(page.getByRole("button")).not.toBeInTheDocument();
});
