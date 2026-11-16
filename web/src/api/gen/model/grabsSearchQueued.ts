import type { GrabsQueuedGroup } from "./grabsQueuedGroup.ts";

export interface GrabsSearchQueued {
  /** Each search group the press queued, in tree order */
  groups: GrabsQueuedGroup[];
}
