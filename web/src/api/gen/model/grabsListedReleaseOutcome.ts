/**
 * grabbed is the run's Grab. not_taken ranked above it, and its Indexer stopped, reached its limit or dropped it. outranked ranked below it. kept ranked, and the run grabbed nothing. refused has a refusal of its own or is grabbable at no cell. waiting is a candidate a cell's Hold keeps until it grabs the best one.
 */
export type GrabsListedReleaseOutcome =
  | "grabbed"
  | "not_taken"
  | "outranked"
  | "kept"
  | "refused"
  | "waiting";
