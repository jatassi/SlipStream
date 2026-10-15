import { expect, onTestFinished, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { frames } from "../test/frames";
import { renderKit } from "../test/render-kit";
import { Popover } from "./popover";

test("a detail popover opens on press, is 340px wide and closes back to its trigger on Esc", async () => {
  renderKit(
    <Popover trigger={<button type="button">Why this release</button>} label="Release score">
      Matched 1080p and the preferred group.
    </Popover>,
  );
  const trigger = page.getByRole("button", { name: "Why this release" });
  await userEvent.click(trigger);
  const popover = page.getByRole("dialog", { name: "Release score" });
  await expect.element(popover).toHaveTextContent("Matched 1080p and the preferred group.");
  expect(getComputedStyle(popover.element()).width).toBe("340px");

  await userEvent.keyboard("{Escape}");
  await expect.element(popover).not.toBeInTheDocument();
  await expect.element(trigger).toHaveFocus();
});

// Base UI's safe polygon waits 40ms before it counts the pointer as gone, and
// the 160ms close delay starts then.
test("a hover popover hides 160ms after the pointer leaves", async () => {
  renderKit(
    <>
      <button type="button">Elsewhere</button>
      <Popover trigger={<button type="button">Seeders</button>} label="Seeders" opensOn="hover">
        42 peers, 12 seeding.
      </Popover>
    </>,
    { density: "touch" },
  );
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  onTestFinished(() => {
    vi.useRealTimers();
  });
  const trigger = page.getByRole("button", { name: "Seeders" });

  await userEvent.hover(trigger);
  await vi.advanceTimersByTimeAsync(300);
  await expect
    .element(page.getByRole("dialog", { name: "Seeders" }))
    .toHaveTextContent("42 peers, 12 seeding.");

  const expanded = async () => {
    await frames(2);
    return trigger.element().getAttribute("aria-expanded");
  };
  await userEvent.hover(page.getByRole("button", { name: "Elsewhere" }));
  await vi.advanceTimersByTimeAsync(40 + 159);
  expect(await expanded()).toBe("true");
  await vi.advanceTimersByTimeAsync(1);
  expect(await expanded()).toBe("false");
});
