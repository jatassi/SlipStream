/**
 * Null while the run searches
 * @nullable
 */
export type GrabsSearchOutcome =
  | "grabbed"
  | "nothing_acceptable"
  | "indexer_unavailable"
  | "indexer_refused"
  | null;
