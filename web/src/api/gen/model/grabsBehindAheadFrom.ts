/**
 * What queued the Works ahead, as in Queued behind 140 Works from Search all; null when none are
 * @nullable
 */
export type GrabsBehindAheadFrom =
  | "search"
  | "search_all"
  | "add"
  | "monitor"
  | "profile"
  | "editions"
  | "retarget"
  | "release_date"
  | "refresh"
  | "backlog"
  | "failed_grab"
  | null;
