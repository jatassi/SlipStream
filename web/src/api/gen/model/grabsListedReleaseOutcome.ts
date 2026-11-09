/**
 * grabbed is the run's Grab. not_taken ranked above it, and its Indexer stopped, reached its limit or dropped it. outranked ranked below it. kept ranked, and the run grabbed nothing. refused has a refusal of its own or is grabbable at no cell.
 */
export type GrabsListedReleaseOutcome = "grabbed" | "not_taken" | "outranked" | "kept" | "refused";
