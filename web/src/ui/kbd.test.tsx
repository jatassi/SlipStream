import { FitToScreenIcon } from "@hugeicons/core-free-icons";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { IconButton } from "./icon-button";
import { type KeyLike, Kbd, chordOf, shortcut } from "./kbd";
import { Tooltip } from "./tooltip";

test("a chord prints as Apple's glyphs on a Mac and as names joined by + elsewhere", () => {
  expect([
    shortcut(["Mod", "Shift", "K"], true),
    shortcut(["Shift", "Mod", "K"], false),
    shortcut(["Escape"], true),
  ]).toEqual([
    { chord: "Mod+Shift+K", keys: "Shift+Meta+K", label: "⇧⌘K" },
    { chord: "Mod+Shift+K", keys: "Control+Shift+K", label: "Ctrl+Shift+K" },
    { chord: "Escape", keys: "Escape", label: "Esc" },
  ]);
});

const press = (key: string, held: Partial<KeyLike> = {}): KeyLike => ({
  key,
  code: "",
  ctrlKey: false,
  metaKey: false,
  altKey: false,
  shiftKey: false,
  isComposing: false,
  ...held,
});

const palette = (apple: boolean) => shortcut(["Mod", "K"], apple).chord;

test("a press matches its shortcut by chord on the viewer's platform", () => {
  expect([
    chordOf(press("k", { metaKey: true }), true) === palette(true),
    chordOf(press("k", { ctrlKey: true }), false) === palette(false),
    chordOf(press("k", { ctrlKey: true }), true) === palette(true),
    chordOf(press("k", { ctrlKey: true, shiftKey: true }), false) === palette(false),
  ]).toEqual([true, true, false, false]);
});

test("a keycap is hidden from screen readers, and a touch screen shows none", async () => {
  renderKit(
    <p>
      Search <Kbd shortcut={shortcut(["Mod", "K"], false)} />
    </p>,
  );
  await expect.element(page.getByText("Ctrl+K")).toHaveAttribute("aria-hidden", "true");

  renderKit(
    <p>
      Open <Kbd shortcut={shortcut(["Enter"])} />
    </p>,
    { shell: "phone" },
  );
  await expect.element(page.getByText("Open")).toBeVisible();
  expect(document.querySelectorAll("kbd")).toHaveLength(1);
});

test("a tooltip names the key that does the same after its words", async () => {
  renderKit(
    <div className="p-10">
      <Tooltip label="Fit" shortcut={shortcut(["0"])}>
        <IconButton icon={FitToScreenIcon} label="Fit" />
      </Tooltip>
    </div>,
  );
  await userEvent.hover(page.getByRole("button", { name: "Fit" }));
  await expect
    .poll(() => page.getByText("Fit", { exact: true }).query()?.parentElement?.textContent)
    .toBe("Fit0");
});
