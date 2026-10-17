import { isProblem } from "../api/errors";
import type { TargetsEditions } from "../api/gen/model";
import { type Cut, type EditionList, standard } from "./editions-sentence";

type Keys = readonly [string, ...string[]];

/**
 * An Edition list being edited. `cuts` is the Choose cuts list, kept while
 * Any cut is chosen so switching back restores it.
 */
export type EditionsDraft = { readonly kind: EditionList["kind"]; readonly cuts: Keys };

export const editionsDraftOf = (editions: TargetsEditions): EditionsDraft => ({
  kind: editions.kind,
  cuts: editions.kind === "cuts" ? keysOf(editions.cuts) : [standard],
});

export const editionsBodyOf = (draft: EditionsDraft): TargetsEditions =>
  draft.kind === "any" ? { kind: "any" } : { kind: "cuts", cuts: [...draft.cuts] };

/**
 * Runs a save of an Edition list, saying plainly when a cut on it was
 * deleted since the list was read.
 */
export async function savingCuts<T>(save: () => Promise<T>): Promise<T> {
  try {
    return await save();
  } catch (error) {
    if (isProblem(error, "cut_not_found")) {
      throw new Error(
        "A cut on this list was deleted, so nothing was saved. Take it off the list and save again.",
        { cause: error },
      );
    }
    throw error;
  }
}

export function keysOf(keys: readonly string[]): Keys {
  const [first, ...rest] = keys;
  if (first === undefined) {
    throw new Error("a Choose cuts list keeps at least one cut");
  }
  return [first, ...rest];
}

export function listOf(draft: EditionsDraft, nameOf: (key: string) => string): EditionList {
  if (draft.kind === "any") {
    return { kind: "any" };
  }
  const cut = (key: string): Cut => ({ key, name: nameOf(key) });
  const [first, ...rest] = draft.cuts;
  return { kind: "cuts", cuts: [cut(first), ...rest.map((key) => cut(key))] };
}
