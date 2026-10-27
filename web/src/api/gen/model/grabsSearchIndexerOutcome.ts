/**
 * Null while its search runs
 * @nullable
 */
export type GrabsSearchIndexerOutcome =
  | "found"
  | "no_releases"
  | "limited"
  | "failing"
  | "stopped"
  | "waiting"
  | "late"
  | "limit_reached"
  | "no_categories"
  | "removed"
  | null;
