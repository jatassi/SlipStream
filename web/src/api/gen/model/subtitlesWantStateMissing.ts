/**
 * Why a missing want has no subtitle yet, the first that applies
 * @nullable
 */
export type SubtitlesWantStateMissing =
  | "no_holder"
  | "no_provider"
  | "paused"
  | "no_aired_number"
  | "map_failed"
  | "map_due"
  | "none_passed"
  | "nothing_found"
  | "never_searched"
  | null;
