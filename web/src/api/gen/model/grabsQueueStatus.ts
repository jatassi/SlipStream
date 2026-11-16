import type { GrabsLimitUntil } from "./grabsLimitUntil.ts";

export interface GrabsQueueStatus {
  /** Each Indexer whose search limit holds queued searches */
  limits: GrabsLimitUntil[];
  /**
   * When Backlog search runs next; null while Backlog search is off
   * @nullable
   */
  nextBacklogAt: string | null;
  /**
   * Search groups queued
   * @minimum 0
   */
  queued: number;
  /**
   * Search groups searching now
   * @minimum 0
   */
  searching: number;
}
