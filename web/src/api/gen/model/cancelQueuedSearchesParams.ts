import type { CancelQueuedSearchesLane } from "./cancelQueuedSearchesLane.ts";

export type CancelQueuedSearchesParams = {
  /**
   * The lane to empty. Only bulk can be canceled
   */
  lane: CancelQueuedSearchesLane;
};
