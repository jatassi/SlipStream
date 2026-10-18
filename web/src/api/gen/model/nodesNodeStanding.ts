/**
 * How the Node stands with the Controller since link.since: in step, reconnecting within the blip, offline, lapsed past the offline grace, or on another version
 */
export type NodesNodeStanding = "in-step" | "reconnecting" | "offline" | "lapsed" | "mismatched";
