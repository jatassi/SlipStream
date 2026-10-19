import {
  type KeyboardEvent,
  type RefObject,
  useEffect,
  useEffectEvent,
  useReducer,
  useState,
} from "react";
import { flushSync } from "react-dom";

import { readPref, writePref } from "../shell/prefs";
import { appleViewer } from "../ui/platform";
import { type Intent, intentOf } from "./browser-keys";
import { type BrowserAction, type BrowserState, browse, startBrowsing } from "./browser-state";
import { type Row, pathOf, rowsOf } from "./rows";
import { type FolderListing, useFolderListing, useSuggestions } from "./use-folder-listing";
import { useNewFolder } from "./use-new-folder";
import { type FolderCheck, type Judging, usable, useVerdict } from "./verdict";

export type BrowserProps = {
  readonly nodeId: string;
  readonly nodeName: string;
  /** "" opens on Places. */
  readonly startPath: string;
  readonly check: FolderCheck;
  readonly onPick: (path: string) => void;
  readonly close: () => void;
  /** The filter or path field, which keeps focus while the list is driven. */
  readonly field: RefObject<HTMLInputElement | null>;
  /** How many rows fit in the list's view, for Page Up and Page Down. */
  readonly page: () => number;
  /** Whether focus returns to the field after an action, as it does with a keyboard. */
  readonly refocus: boolean;
};

type Navigation = {
  readonly state: BrowserState;
  readonly dispatch: (action: BrowserAction) => void;
  readonly listing: FolderListing;
  readonly rows: readonly Row[];
  readonly open: (path: string, cameFrom?: string) => void;
  readonly up: () => void;
  readonly choose: (index: number) => void;
  readonly toggleHidden: () => void;
  readonly focusField: () => void;
};

export type Browser = Omit<Navigation, "dispatch" | "focusField"> & {
  readonly judging: Judging;
  /** The New folder row's folder is being made. */
  readonly making: boolean;
  /** The name of the folder on show: its last breadcrumb, or the Node's on Places. */
  readonly name: string;
  readonly type: (text: string) => void;
  readonly newFolder: () => void;
  readonly nameFolder: (name: string) => void;
  readonly cancelNewFolder: () => void;
  readonly create: () => void;
  readonly showPath: () => void;
  readonly pick: () => void;
  readonly retry: () => void;
  readonly onKeyDown: (
    event: KeyboardEvent<HTMLInputElement>,
    from: "field" | "folderName",
  ) => void;
};

// The rows on show re-anchor the active one, so the reducer hears of each change once.
function useRelisted(
  state: BrowserState,
  rows: readonly Row[],
  loading: boolean,
  dispatch: Navigation["dispatch"],
) {
  const paths = rows.map((row) => pathOf(row) ?? "");
  const listed = loading ? null : JSON.stringify([state.path, ...paths]);
  const relist = useEffectEvent(() => {
    dispatch({ type: "listed", path: state.path, paths });
  });
  useEffect(() => {
    if (listed !== null) {
      relist();
    }
  }, [listed]);
}

function useNavigation({ nodeId, startPath, field, refocus }: BrowserProps): Navigation {
  const [state, dispatch] = useReducer(browse, undefined, () =>
    startBrowsing(startPath, readPref("showHiddenFolders")),
  );
  // Where Go up leads while the folder asked for has no listing to name its parent.
  const [back, setBack] = useState("");
  const listing = useFolderListing(nodeId, state.path, state.showHidden);
  const rows = rowsOf(state, listing.shown, useSuggestions(nodeId, state.field));
  useRelisted(state, rows, listing.loading, dispatch);
  const { current } = listing;
  const focusField = () => {
    if (refocus) {
      field.current?.focus();
    }
  };
  const open = (path: string, cameFrom?: string) => {
    setBack(current?.path ?? back);
    dispatch({ type: "open", path, ...(cameFrom !== undefined && { cameFrom }) });
    focusField();
  };
  const toggleHidden = () => {
    writePref("showHiddenFolders", !state.showHidden);
    dispatch({ type: "toggleHidden" });
  };
  return {
    state,
    dispatch,
    listing,
    rows,
    open,
    up: () => {
      if (current === undefined) {
        open(back);
      } else if (current.path !== "") {
        open(current.parent, current.path);
      }
    },
    choose: (index) => {
      const row = rows[index];
      const path = row === undefined ? undefined : pathOf(row);
      if (path === null) {
        toggleHidden();
      } else if (path !== undefined) {
        open(path);
      }
    },
    toggleHidden,
    focusField,
  };
}

function run(
  intent: Intent,
  browser: Omit<Browser, "onKeyDown">,
  dispatch: Navigation["dispatch"],
  close: () => void,
) {
  switch (intent.type) {
    case "move":
    case "clear":
    case "newFolder":
      dispatch(intent);
      return;
    case "toggleHidden":
      browser.toggleHidden();
      return;
    case "cancelNewFolder":
      browser.cancelNewFolder();
      return;
    case "openActive":
    case "complete":
      browser.choose(browser.state.active);
      return;
    case "up":
      browser.up();
      return;
    case "use":
      browser.pick();
      return;
    case "go":
      browser.open(browser.state.field);
      return;
    case "showPath":
      browser.showPath();
      return;
    case "close":
      close();
      return;
    case "create":
      browser.create();
      return;
    default:
      intent satisfies never;
  }
}

function edits(
  dispatch: Navigation["dispatch"],
  focusField: () => void,
  field: BrowserProps["field"],
  current: string | undefined,
) {
  return {
    type: (text: string) => {
      dispatch({ type: "type", text });
    },
    newFolder: () => {
      dispatch({ type: "newFolder" });
    },
    nameFolder: (name: string) => {
      dispatch({ type: "nameFolder", name });
    },
    cancelNewFolder: () => {
      dispatch({ type: "cancelNewFolder" });
      focusField();
    },
    showPath: () => {
      if (current === undefined || current === "") {
        return;
      }
      flushSync(() => {
        dispatch({ type: "type", text: current });
      });
      field.current?.focus();
      field.current?.select();
    },
  };
}

// Keys that clear or cancel inside the browser would otherwise close it too.
const keptInside = new Set<Intent["type"]>(["clear", "cancelNewFolder", "close"]);

function keyHandler(
  browser: Omit<Browser, "onKeyDown">,
  dispatch: Navigation["dispatch"],
  close: () => void,
  page: () => number,
): Browser["onKeyDown"] {
  return (event, from) => {
    const intent = intentOf(event.nativeEvent, {
      mac: appleViewer,
      state: browser.state,
      rows: browser.rows.length,
      page: page(),
      in: from,
    });
    if (intent === null) {
      return;
    }
    event.preventDefault();
    if (keptInside.has(intent.type)) {
      event.stopPropagation();
    }
    run(intent, browser, dispatch, close);
  };
}

export function useBrowser(props: BrowserProps): Browser {
  const { nodeId, nodeName, check, onPick, close, field, page } = props;
  const { dispatch, focusField, ...navigation } = useNavigation(props);
  const { state, listing } = navigation;
  const { current } = listing;
  const judging = useVerdict(check, nodeId, current?.path ?? "");
  const folder = useNewFolder({
    nodeId,
    nodeName,
    parent: current?.path,
    creating: state.creating,
    dispatch,
    open: navigation.open,
  });
  const browser: Omit<Browser, "onKeyDown"> = {
    ...navigation,
    ...edits(dispatch, focusField, field, current?.path),
    judging,
    making: folder.busy,
    name: current?.segments.at(-1)?.name ?? nodeName,
    create: folder.create,
    pick: () => {
      if (current !== undefined && usable(judging)) {
        onPick(current.path);
        close();
      }
    },
    retry: () => {
      listing.retry();
      focusField();
    },
  };
  return { ...browser, onKeyDown: keyHandler(browser, dispatch, close, page) };
}
