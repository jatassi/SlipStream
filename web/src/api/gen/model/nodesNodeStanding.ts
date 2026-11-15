/**
 * The Node's connection to the Controller since link.since: in-step (connected), reconnecting (a short drop), offline, lapsed (offline longer than the offline grace) or mismatched (another version)
 */
export type NodesNodeStanding = "in-step" | "reconnecting" | "offline" | "lapsed" | "mismatched";
