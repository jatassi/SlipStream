/**
 * home_only: the Indexer stopped while searching from a failover Node, and searches only from its home Node until that Node is back
 */
export type IndexersStatusState = "ok" | "limited" | "failing" | "stopped" | "home_only";
