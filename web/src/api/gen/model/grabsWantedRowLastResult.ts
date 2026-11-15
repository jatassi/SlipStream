/**
 * How its last search finished; null before one did
 * @nullable
 */
export type GrabsWantedRowLastResult =
  | "grabbed"
  | "waiting"
  | "nothing_found"
  | "nothing_acceptable"
  | "no_indexer"
  | "no_root"
  | null;
