import type { ConversionsLaneRowsLane } from "./conversionsLaneRowsLane.ts";
import type { ConversionsUpNext } from "./conversionsUpNext.ts";

export interface ConversionsLaneRows {
  lane: ConversionsLaneRowsLane;
  /** In run order */
  rows: ConversionsUpNext[];
}
