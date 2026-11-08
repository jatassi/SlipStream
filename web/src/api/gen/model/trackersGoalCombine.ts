/**
 * first is whichever of the ratio and seeding time is reached first; both needs both of them; forever seeds until an Admin removes the torrent
 */
export type TrackersGoalCombine = "first" | "both" | "forever";
