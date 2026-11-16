import type { GrabsBehindAheadFrom } from "./grabsBehindAheadFrom.ts";
import type { GrabsBehindAheadLane } from "./grabsBehindAheadLane.ts";

export interface GrabsBehind {
  /**
   * Works ahead of it in its lane
   * @minimum 0
   */
  ahead: number;
  /**
   * What queued the Works ahead, as in Queued behind 140 Works from Search all; null when none are
   * @nullable
   */
  aheadFrom: GrabsBehindAheadFrom;
  /**
   * The lane of the Works ahead; null when none are
   * @nullable
   */
  aheadLane: GrabsBehindAheadLane;
  /**
   * The Indexer whose search limit it waits for; null when it waits for none
   * @nullable
   */
  indexerId: string | null;
  /**
   * That Indexer's name; null when it waits for none
   * @nullable
   */
  indexerName: string | null;
  /** RSS sync uses all of the Indexer's search limit, so Backlog search and Search all wait for it */
  starved: boolean;
  /**
   * When the Indexer's search limit frees a search; null when it waits for none
   * @nullable
   */
  until: string | null;
}
