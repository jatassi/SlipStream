/**
 * The queued entry's lane, else the running search's; null while idle
 * @nullable
 */
export type GrabsGroupStateLane = "now" | "bulk" | "backlog" | null;
