import { type Move, moveIndex } from "../ui/move-index";

export type NewFolder = { readonly name: string; readonly error: string | null };

/**
 * The folder browser's whole interaction. Paths are the Node's own strings:
 * the browser compares them for equality and never splits or joins them.
 */
export type BrowserState = {
  /** "" is Places. */
  readonly path: string;
  /** Filter text, or a typed path. */
  readonly field: string;
  /** An index into the rows on show; anything outside them is none. */
  readonly active: number;
  /** The folder just left by going up, made active once its parent lists. */
  readonly cameFrom: string | null;
  readonly showHidden: boolean;
  readonly creating: NewFolder | null;
};

export type BrowserAction =
  | { readonly type: "open"; readonly path: string; readonly cameFrom?: string }
  | { readonly type: "type"; readonly text: string }
  | { readonly type: "move"; readonly to: Move; readonly count: number; readonly page: number }
  /** The paths of the rows now on show for `path`. */
  | { readonly type: "listed"; readonly path: string; readonly paths: readonly string[] }
  | { readonly type: "toggleHidden" }
  | { readonly type: "newFolder" }
  | { readonly type: "nameFolder"; readonly name: string }
  | { readonly type: "folderRefused"; readonly error: string }
  | { readonly type: "cancelNewFolder" }
  | { readonly type: "clear" };

export const newFolderName = "New folder";

export const startBrowsing = (path: string, showHidden: boolean): BrowserState => ({
  path,
  field: "",
  active: -1,
  cameFrom: null,
  showHidden,
  creating: null,
});

function anchor(state: BrowserState, paths: readonly string[]): number {
  if (paths.length === 0) {
    return -1;
  }
  const from = state.cameFrom === null ? -1 : paths.indexOf(state.cameFrom);
  if (from >= 0) {
    return from;
  }
  return state.active >= 0 && state.active < paths.length ? state.active : 0;
}

export function browse(state: BrowserState, action: BrowserAction): BrowserState {
  switch (action.type) {
    case "open":
      return {
        ...startBrowsing(action.path, state.showHidden),
        cameFrom: action.cameFrom ?? null,
      };
    case "type":
      return { ...state, field: action.text, active: 0, cameFrom: null };
    case "move":
      return { ...state, active: moveIndex(state.active, action.to, action.count, action.page) };
    case "listed":
      if (action.path !== state.path) {
        return state;
      }
      return { ...state, active: anchor(state, action.paths), cameFrom: null };
    case "toggleHidden":
      return { ...state, showHidden: !state.showHidden };
    case "newFolder":
      if (state.path === "" || state.creating !== null) {
        return state;
      }
      return { ...state, field: "", creating: { name: newFolderName, error: null } };
    case "nameFolder":
      if (state.creating === null) {
        return state;
      }
      return { ...state, creating: { name: action.name, error: null } };
    case "folderRefused":
      if (state.creating === null) {
        return state;
      }
      return { ...state, creating: { ...state.creating, error: action.error } };
    case "cancelNewFolder":
      return { ...state, creating: null };
    case "clear":
      return { ...state, field: "", active: 0 };
    default:
      return action satisfies never;
  }
}

/**
 * Whether the field holds a path rather than a filter: a `/` or `\` anywhere
 * (which covers `\\server`), or a leading drive such as `C:`. It detects only;
 * the Node splits and resolves the path in its own dialect.
 */
export const isPath = (text: string): boolean => /[/\\]|^[a-z]:/iu.test(text);
