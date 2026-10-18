import { describe, expect, test } from "vitest";

import { type Intent, type KeyContext, type KeyLike, intentOf } from "./browser-keys";
import type { Move } from "../ui/move-index";
import { type BrowserState, startBrowsing } from "./browser-state";

type Mods = { ctrl?: boolean; meta?: boolean; alt?: boolean; shift?: boolean };

const press = (
  key: string,
  { ctrl = false, meta = false, alt = false, shift = false }: Mods = {},
  code = "",
): KeyLike => ({
  key,
  code,
  ctrlKey: ctrl,
  metaKey: meta,
  altKey: alt,
  shiftKey: shift,
  isComposing: false,
});

const folder = (changes: Partial<BrowserState> = {}): BrowserState => ({
  ...startBrowsing(String.raw`D:\Media\TV`, false),
  active: 0,
  ...changes,
});

const context = (mac: boolean, changes: Partial<KeyContext> = {}): KeyContext => ({
  mac,
  state: folder(),
  rows: 50,
  page: 10,
  in: "field",
  ...changes,
});

const move = (to: Move): Intent => ({ type: "move", to, count: 50, page: 10 });

const platforms = [
  { name: "a Mac viewer", mac: true, mod: { meta: true } },
  { name: "another viewer", mac: false, mod: { ctrl: true } },
] as const;

describe.each(platforms)("moving for $name", ({ mac }) => {
  const at = (changes: Partial<KeyContext> = {}) => context(mac, changes);

  test.each([
    ["ArrowDown", "next"],
    ["ArrowUp", "prev"],
    ["PageDown", "pageDown"],
    ["PageUp", "pageUp"],
    ["Home", "first"],
    ["End", "last"],
  ] as const)("%s moves the active row %s", (key, to) => {
    expect(intentOf(press(key), at())).toEqual(move(to));
  });

  test("moving keys pass through when no rows show", () => {
    for (const key of ["ArrowDown", "ArrowUp", "PageDown", "PageUp", "Home", "End"]) {
      expect(intentOf(press(key), at({ rows: 0 }))).toBeNull();
    }
  });

  test("Home and End move the caret once the field has text", () => {
    const typed = at({ state: folder({ field: "an" }) });
    expect(intentOf(press("Home"), typed)).toBeNull();
    expect(intentOf(press("End"), typed)).toBeNull();
    expect(intentOf(press("ArrowDown"), typed)).toEqual(move("next"));
  });
});

describe.each(platforms)("opening and going up for $name", ({ mac }) => {
  const at = (changes: Partial<KeyContext> = {}) => context(mac, changes);

  test("Enter opens the active row, or goes to a typed path", () => {
    expect(intentOf(press("Enter"), at())).toEqual({ type: "openActive" });
    expect(intentOf(press("Enter"), at({ state: folder({ field: "an" }) }))).toEqual({
      type: "openActive",
    });
    expect(intentOf(press("Enter"), at({ state: folder({ field: String.raw`D:\Me` }) }))).toEqual({
      type: "go",
    });
    expect(intentOf(press("Enter"), at({ state: folder({ field: "/mnt" }), rows: 0 }))).toEqual({
      type: "go",
    });
    expect(intentOf(press("Enter"), at({ rows: 0 }))).toBeNull();
  });

  test("→ opens the active row only with an empty field", () => {
    expect(intentOf(press("ArrowRight"), at())).toEqual({ type: "openActive" });
    expect(intentOf(press("ArrowRight"), at({ state: folder({ field: "an" }) }))).toBeNull();
    expect(intentOf(press("ArrowRight"), at({ rows: 0 }))).toBeNull();
  });

  test("Backspace goes up only from an empty field inside a folder", () => {
    expect(intentOf(press("Backspace"), at())).toEqual({ type: "up" });
    expect(intentOf(press("Backspace"), at({ state: folder({ field: "a" }) }))).toBeNull();
    expect(intentOf(press("Backspace"), at({ state: folder({ path: "" }) }))).toBeNull();
  });

  test("Tab completes the active suggestion of a typed path, and is Tab otherwise", () => {
    const typing = at({ state: folder({ field: "/mnt/me" }) });
    expect(intentOf(press("Tab"), typing)).toEqual({ type: "complete" });
    expect(intentOf(press("Tab"), { ...typing, rows: 0 })).toBeNull();
    expect(intentOf(press("Tab"), at({ state: folder({ field: "an" }) }))).toBeNull();
    expect(intentOf(press("Tab", { shift: true }), typing)).toBeNull();
  });
});

describe.each(platforms)("Esc and shortcuts for $name", ({ mac, mod }) => {
  const at = (changes: Partial<KeyContext> = {}) => context(mac, changes);

  test("Esc clears the field, then cancels New folder, then closes", () => {
    const creating = { name: "New folder", error: null };
    expect(intentOf(press("Escape"), at({ state: folder({ field: "an", creating }) }))).toEqual({
      type: "clear",
    });
    expect(intentOf(press("Escape"), at({ state: folder({ creating }) }))).toEqual({
      type: "cancelNewFolder",
    });
    expect(intentOf(press("Escape"), at())).toEqual({ type: "close" });
  });

  test("the primary modifier with Enter uses the folder, except on Places", () => {
    expect(intentOf(press("Enter", mod), at())).toEqual({ type: "use" });
    expect(intentOf(press("Enter", mod), at({ state: folder({ path: "" }) }))).toBeNull();
  });

  test("the primary modifier with L puts the full path in the field", () => {
    expect(intentOf(press("l", mod), at())).toEqual({ type: "showPath" });
    expect(intentOf(press("l", mod), at({ state: folder({ path: "" }) }))).toBeNull();
  });

  test("the primary modifier with Shift+N adds a New folder row, except on Places", () => {
    expect(intentOf(press("N", { ...mod, shift: true }), at())).toEqual({ type: "newFolder" });
    expect(
      intentOf(press("N", { ...mod, shift: true }), at({ state: folder({ path: "" }) })),
    ).toBeNull();
  });
});

describe.each(platforms)("the New folder row and other keys for $name", ({ mac, mod }) => {
  const at = (changes: Partial<KeyContext> = {}) => context(mac, changes);

  test("in the New folder row, Enter creates and Esc cancels; other keys stay the row's", () => {
    const naming = at({
      in: "folderName",
      state: folder({ creating: { name: "Mo", error: null } }),
    });
    expect(intentOf(press("Enter"), naming)).toEqual({ type: "create" });
    expect(intentOf(press("Escape"), naming)).toEqual({ type: "cancelNewFolder" });
    for (const key of ["ArrowDown", "Home", "Backspace", "ArrowRight", "Tab"]) {
      expect(intentOf(press(key), naming)).toBeNull();
    }
    expect(intentOf(press("Enter", mod), naming)).toBeNull();
  });

  test("keys typed mid-composition stay the input method's", () => {
    expect(intentOf({ ...press("Enter"), isComposing: true }, at())).toBeNull();
  });

  test("letters with no binding stay the field's", () => {
    expect(intentOf(press("a"), at())).toBeNull();
    expect(intentOf(press("A", { shift: true }), at())).toBeNull();
    expect(intentOf(press("N", { shift: true }), at())).toBeNull();
  });
});

const mac = (changes: Partial<KeyContext> = {}) => context(true, changes);
const other = (changes: Partial<KeyContext> = {}) => context(false, changes);

describe("a Mac viewer", () => {
  test("goes up with ⌘↑", () => {
    expect(intentOf(press("ArrowUp", { meta: true }), mac())).toEqual({ type: "up" });
    expect(
      intentOf(press("ArrowUp", { meta: true }), mac({ state: folder({ path: "" }) })),
    ).toBeNull();
    expect(intentOf(press("ArrowUp", { alt: true }), mac())).toBeNull();
  });

  test("toggles Show hidden with ⌘⇧.", () => {
    expect(intentOf(press(">", { meta: true, shift: true }, "Period"), mac())).toEqual({
      type: "toggleHidden",
    });
    expect(intentOf(press("h", { meta: true }), mac())).toBeNull();
  });

  test("takes no Ctrl shortcuts, F4 or Alt+D", () => {
    expect(intentOf(press("Enter", { ctrl: true }), mac())).toBeNull();
    expect(intentOf(press("l", { ctrl: true }), mac())).toBeNull();
    expect(intentOf(press("F4"), mac())).toBeNull();
    expect(intentOf(press("d", { alt: true }), mac())).toBeNull();
  });
});

describe("another viewer", () => {
  test("goes up with Alt+↑, even with text in the field", () => {
    expect(intentOf(press("ArrowUp", { alt: true }), other())).toEqual({ type: "up" });
    expect(
      intentOf(press("ArrowUp", { alt: true }), other({ state: folder({ field: "an" }) })),
    ).toEqual({ type: "up" });
    expect(
      intentOf(press("ArrowUp", { alt: true }), other({ state: folder({ path: "" }) })),
    ).toBeNull();
  });

  test("puts the full path in the field with F4 or Alt+D too", () => {
    expect(intentOf(press("F4"), other())).toEqual({ type: "showPath" });
    expect(intentOf(press("d", { alt: true }), other())).toEqual({ type: "showPath" });
  });

  test("toggles Show hidden with Ctrl+H, even on Places", () => {
    expect(intentOf(press("h", { ctrl: true }), other())).toEqual({ type: "toggleHidden" });
    expect(intentOf(press("h", { ctrl: true }), other({ state: folder({ path: "" }) }))).toEqual({
      type: "toggleHidden",
    });
  });

  test("takes no ⌘ shortcuts", () => {
    expect(intentOf(press("Enter", { meta: true }), other())).toBeNull();
    expect(intentOf(press("ArrowUp", { meta: true }), other())).toBeNull();
    expect(intentOf(press(">", { meta: true, shift: true }, "Period"), other())).toBeNull();
  });
});
