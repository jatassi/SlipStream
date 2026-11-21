import { appleViewer } from "./platform";
import { useDensity } from "./shell-context";

/** A native KeyboardEvent fits; a React one fits through its nativeEvent. */
export type KeyLike = {
  readonly key: string;
  readonly code: string;
  readonly ctrlKey: boolean;
  readonly metaKey: boolean;
  readonly altKey: boolean;
  readonly shiftKey: boolean;
  readonly isComposing: boolean;
};

/** "Mod" is the viewer's primary modifier: ⌘ on Apple platforms, Ctrl elsewhere. */
const modifiers = {
  Mod: { apple: "⌘", other: "Ctrl", aria: { apple: "Meta", other: "Control" } },
  Alt: { apple: "⌥", other: "Alt", aria: { apple: "Alt", other: "Alt" } },
  Shift: { apple: "⇧", other: "Shift", aria: { apple: "Shift", other: "Shift" } },
} as const;

type Modifier = keyof typeof modifiers;

type Platform = "apple" | "other";

// Apple prints its modifiers in ⌃⌥⇧⌘ order, and everyone else names Ctrl first.
const order: { readonly [P in Platform]: readonly Modifier[] } = {
  apple: ["Alt", "Shift", "Mod"],
  other: ["Mod", "Alt", "Shift"],
};

/** Each key a hint shows, by its KeyboardEvent name, with the glyph its keycap prints. */
const glyphs = {
  ArrowUp: "↑",
  ArrowDown: "↓",
  Enter: "↵",
  Escape: "Esc",
  K: "K",
  "+": "+",
  "-": "-",
  "0": "0",
} as const;

export type Key = Modifier | keyof typeof glyphs;

/** A key or chord, in the three spellings a control needs. */
export type Shortcut = {
  /** "Mod+Shift+N", what chordOf returns for its press. */
  readonly chord: string;
  /** "Meta+Shift+N", for aria-keyshortcuts and a test's key presses. */
  readonly keys: string;
  /** "⇧⌘N" on Apple platforms, "Ctrl+Shift+N" elsewhere. */
  readonly label: string;
};

const isModifier = (key: Key): key is Modifier => key in modifiers;

export function shortcut(pressed: readonly Key[], apple: boolean = appleViewer): Shortcut {
  const platform: Platform = apple ? "apple" : "other";
  const held = (on: Platform) => order[on].filter((modifier) => pressed.includes(modifier));
  const named = pressed.filter((key) => !isModifier(key));
  return {
    chord: [...held("other"), ...named].join("+"),
    keys: [...held(platform).map((modifier) => modifiers[modifier].aria[platform]), ...named].join(
      "+",
    ),
    label: [
      ...held(platform).map((modifier) => modifiers[modifier][platform]),
      ...named.map((key) => glyphs[key]),
    ].join(apple ? "" : "+"),
  };
}

/** A press as a chord, "Mod+Shift+N", with the viewer's primary modifier as Mod. */
export function chordOf(key: KeyLike, apple: boolean = appleViewer): string {
  const primary = apple ? key.metaKey : key.ctrlKey;
  const secondary = apple ? key.ctrlKey : key.metaKey;
  const name = key.code === "Period" ? "." : key.key.length === 1 ? key.key.toUpperCase() : key.key;
  return [
    primary && "Mod",
    secondary && (apple ? "Ctrl" : "Meta"),
    key.altKey && "Alt",
    key.shiftKey && "Shift",
    name,
  ]
    .filter(Boolean)
    .join("+");
}

/**
 * The key that triggers a control, as one keycap at its trailing edge. Screen
 * readers skip it, since the control carries the same keys in aria-keyshortcuts,
 * and a touch screen shows none, having no keyboard to press.
 */
export function Kbd({ shortcut: shown }: { readonly shortcut: Shortcut }) {
  if (useDensity() === "touch") {
    return null;
  }
  return (
    <kbd
      aria-hidden="true"
      className="shrink-0 rounded-chip bg-fill px-1.5 py-0.5 font-sans text-label text-ink-2"
    >
      {shown.label}
    </kbd>
  );
}
