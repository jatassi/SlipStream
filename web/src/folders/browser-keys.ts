import type { Move } from "../ui/move-index";
import { type BrowserAction, type BrowserState, isPath } from "./browser-state";

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

/**
 * What a key asks of the browser. A key with an intent is the browser's, so
 * its default is prevented; null leaves the key to the input.
 */
export type Intent =
  | Extract<
      BrowserAction,
      { type: "move" | "clear" | "toggleHidden" | "newFolder" | "cancelNewFolder" }
    >
  | { readonly type: "openActive" }
  /** Open the parent folder, re-anchoring on this one. */
  | { readonly type: "up" }
  /** Pick the folder the browser is in. */
  | { readonly type: "use" }
  /** Go to the path typed in the field. */
  | { readonly type: "go" }
  /** Complete the field with the active suggestion and open it. */
  | { readonly type: "complete" }
  /** Put the current folder's full path in the field, selected. */
  | { readonly type: "showPath" }
  | { readonly type: "close" }
  /** Create the New folder row's folder. */
  | { readonly type: "create" };

export type KeyContext = {
  /** The viewer's platform, never the Node's. */
  readonly mac: boolean;
  readonly state: BrowserState;
  /** How many rows are on show. */
  readonly rows: number;
  /** How many rows fit in the list's view. */
  readonly page: number;
  /** Which input has focus: the filter or path field, or the New folder row. */
  readonly in: "field" | "folderName";
};

type Resolve = (context: KeyContext) => Intent | null;

const hasActive = ({ state, rows }: KeyContext) => state.active >= 0 && state.active < rows;
const inFolder = ({ state }: KeyContext) => state.path !== "";
const fieldEmpty = ({ state }: KeyContext) => state.field === "";

const move =
  (to: Move): Resolve =>
  ({ rows, page }) =>
    rows > 0 ? { type: "move", to, count: rows, page } : null;

const when =
  (test: (context: KeyContext) => boolean, intent: Intent): Resolve =>
  (context) =>
    test(context) ? intent : null;

const up = when(inFolder, { type: "up" });
const showPath = when(inFolder, { type: "showPath" });
const toggleHidden: Resolve = () => ({ type: "toggleHidden" });

// Chords name the viewer's primary modifier "Mod": ⌘ on a Mac, Ctrl elsewhere.
const field: Readonly<Record<string, Resolve>> = {
  ArrowDown: move("next"),
  ArrowUp: move("prev"),
  PageDown: move("pageDown"),
  PageUp: move("pageUp"),
  Home: (context) => (fieldEmpty(context) ? move("first")(context) : null),
  End: (context) => (fieldEmpty(context) ? move("last")(context) : null),
  Enter: (context) => {
    if (isPath(context.state.field)) {
      return { type: "go" };
    }
    return hasActive(context) ? { type: "openActive" } : null;
  },
  ArrowRight: when((context) => fieldEmpty(context) && hasActive(context), { type: "openActive" }),
  Backspace: when((context) => fieldEmpty(context) && inFolder(context), { type: "up" }),
  Tab: when((context) => isPath(context.state.field) && hasActive(context), { type: "complete" }),
  Escape: ({ state }) => {
    if (state.field !== "") {
      return { type: "clear" };
    }
    return state.creating === null ? { type: "close" } : { type: "cancelNewFolder" };
  },
  "Mod+Enter": when(inFolder, { type: "use" }),
  "Mod+L": showPath,
  "Mod+Shift+N": when(inFolder, { type: "newFolder" }),
};

const others: Readonly<Record<string, Resolve>> = {
  "Alt+ArrowUp": up,
  F4: showPath,
  "Alt+D": showPath,
  "Mod+H": toggleHidden,
};

// macOS hides the app on ⌘H before a page sees it, so a Mac viewer toggles
// hidden folders with Finder's ⌘⇧. and goes up with Finder's ⌘↑.
const macs: Readonly<Record<string, Resolve>> = {
  "Mod+ArrowUp": up,
  "Mod+Shift+.": toggleHidden,
};

const folderName: Readonly<Record<string, Resolve>> = {
  Enter: () => ({ type: "create" }),
  Escape: () => ({ type: "cancelNewFolder" }),
};

function chordOf(key: KeyLike, mac: boolean): string {
  const primary = mac ? key.metaKey : key.ctrlKey;
  const secondary = mac ? key.ctrlKey : key.metaKey;
  const name = key.code === "Period" ? "." : key.key.length === 1 ? key.key.toUpperCase() : key.key;
  return [
    primary && "Mod",
    secondary && (mac ? "Ctrl" : "Meta"),
    key.altKey && "Alt",
    key.shiftKey && "Shift",
    name,
  ]
    .filter(Boolean)
    .join("+");
}

export function intentOf(key: KeyLike, context: KeyContext): Intent | null {
  if (key.isComposing) {
    return null;
  }
  const chord = chordOf(key, context.mac);
  const resolve =
    context.in === "folderName"
      ? folderName[chord]
      : (field[chord] ?? (context.mac ? macs : others)[chord]);
  return resolve === undefined ? null : resolve(context);
}
