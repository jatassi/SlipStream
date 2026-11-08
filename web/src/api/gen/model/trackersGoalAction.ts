/**
 * What happens to the torrent once the goal is met, where remove-delete also deletes its download copy; null when combine is forever
 * @nullable
 */
export type TrackersGoalAction = "pause" | "remove" | "remove-delete" | null;
