/**
 * first is whichever of the ratio and seeding time is reached first; both needs both of them; forever seeds until an Admin removes the torrent. A goal that isn't forever sets a ratio, a seeding time or both. With only one of them set, first and both wait for that one
 */
export type TrackersGoalCombine = "first" | "both" | "forever";
