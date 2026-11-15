/**
 * What queued it, else what started the running search; null while idle
 * @nullable
 */
export type GrabsGroupStateTrigger =
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
