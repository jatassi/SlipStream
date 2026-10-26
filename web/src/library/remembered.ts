import { readPref, writePref } from "../shell/prefs";
import { type LibrarySearch, librarySearch, viewKeys } from "./view-codec";

/** A Library's id in the remembered views: a Content Type's id, or "all". */
export type ViewScope = string;

/**
 * The view to open on when the address names none: the one last used for
 * this Library in this browser, with the address's search kept.
 */
export function rememberedSearch(scope: ViewScope, search: LibrarySearch): LibrarySearch | undefined {
  if (viewKeys(search).length > 0) {
    return undefined;
  }
  const stored = librarySearch(readPref("libraryViews")[scope]);
  if (viewKeys(stored).length === 0) {
    return undefined;
  }
  const { q } = search;
  return { ...stored, ...(q !== undefined && { q }) };
}

/** Keeps a Library's view for its next visit in this browser; its search isn't kept. */
export function rememberView(scope: ViewScope, search: LibrarySearch): void {
  const { q: _, ...view } = search;
  writePref("libraryViews", { ...readPref("libraryViews"), [scope]: view });
}
