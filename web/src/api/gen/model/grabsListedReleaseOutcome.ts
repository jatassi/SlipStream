/**
 * grabbed is the run's Grab. not_taken ranked above it, but its Indexer had stopped, reached its limit or no longer listed it. outranked ranked below it. kept ranked, but the run grabbed nothing. refused has a refusal of its own, or no Target accepts it. waiting is a candidate SlipStream keeps during a Target's Hold, until it grabs the best one.
 */
export type GrabsListedReleaseOutcome =
  | "grabbed"
  | "not_taken"
  | "outranked"
  | "kept"
  | "refused"
  | "waiting";
