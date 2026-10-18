/**
 * How the home Node stands, out of step within the offline grace: until the grace passes, its searches finish indexer_unavailable
 */
export type IndexersWaitingStanding = "reconnecting" | "offline" | "mismatched";
