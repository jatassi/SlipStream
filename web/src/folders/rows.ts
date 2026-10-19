import type { RootsFolder, RootsListing, RootsPlace } from "../api/gen/model";
import { type BrowserState, isPath } from "./browser-state";

/** One row of the browser's list: what the field and the folder on show offer. */
export type Row =
  | { readonly kind: "place"; readonly place: RootsPlace }
  | { readonly kind: "folder"; readonly folder: RootsFolder }
  | { readonly kind: "suggestion"; readonly path: string }
  /** The trailing row that counts the folders Show hidden would add. */
  | { readonly kind: "hidden"; readonly count: number };

/** The folder a row opens; the hidden row opens none. */
export function pathOf(row: Row): string | null {
  switch (row.kind) {
    case "place":
      return row.place.path;
    case "folder":
      return row.folder.path;
    case "suggestion":
      return row.path;
    case "hidden":
      return null;
    default:
      return row satisfies never;
  }
}

export const rowKey = (row: Row): string => `${row.kind}:${pathOf(row) ?? ""}`;

/** Names starting with the text first, then names containing it, each in the Node's order. */
function matching<T>(items: readonly T[], nameOf: (item: T) => string, text: string): T[] {
  const wanted = text.toLocaleLowerCase();
  if (wanted === "") {
    return [...items];
  }
  const starts: T[] = [];
  const contains: T[] = [];
  for (const item of items) {
    const name = nameOf(item).toLocaleLowerCase();
    if (name.startsWith(wanted)) {
      starts.push(item);
    } else if (name.includes(wanted)) {
      contains.push(item);
    }
  }
  return [...starts, ...contains];
}

export function rowsOf(
  state: BrowserState,
  listing: RootsListing | undefined,
  suggestions: readonly string[],
): Row[] {
  if (isPath(state.field)) {
    return suggestions.map((path) => ({ kind: "suggestion", path }));
  }
  if (listing === undefined) {
    return [];
  }
  if (listing.path === "") {
    return matching(listing.places, (place) => place.name, state.field).map((place) => ({
      kind: "place",
      place,
    }));
  }
  const rows: Row[] = matching(listing.folders, (folder) => folder.name, state.field).map(
    (folder) => ({ kind: "folder", folder }),
  );
  if (state.field === "" && listing.hidden > 0) {
    rows.push({ kind: "hidden", count: listing.hidden });
  }
  return rows;
}
