/**
 * What started it; null for a Manual search's run
 * @nullable
 */
export type GrabsSearchTrigger =
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
  | "rss"
  | null;
