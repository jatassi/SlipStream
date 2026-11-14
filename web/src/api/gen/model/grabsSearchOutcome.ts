/**
 * Null while the run searches. waiting holds a cell for a better Release. nothing_found is a run that found no Release, and nothing_acceptable one whose Releases fit no Target. no_indexer is a run that no Indexer responded to, and no_root one whose Work had no reachable root. listed is a Manual search's run, which takes nothing.
 * @nullable
 */
export type GrabsSearchOutcome =
  | "grabbed"
  | "waiting"
  | "nothing_found"
  | "nothing_acceptable"
  | "no_indexer"
  | "no_root"
  | "listed"
  | null;
