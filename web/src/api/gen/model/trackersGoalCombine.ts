/**
 * first is whichever of the ratio and seeding time is reached first; both needs both of them; forever seeds until an Admin removes the torrent. A goal that ends counts a ratio, a seeding time or both, and with only one of them first and both each wait for it
 */
export type TrackersGoalCombine = "first" | "both" | "forever";
