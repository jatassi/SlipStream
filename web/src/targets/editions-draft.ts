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
  cuts: [standard],
});

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
