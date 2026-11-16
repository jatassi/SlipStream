import type { GrabsQueuedGroupState } from "./grabsQueuedGroupState.ts";

export interface GrabsQueuedGroup {
  /** The group's wanted Items the press searches, in tree order */
  itemIds: string[];
  /** The search group's key, to compare and never to parse */
  key: string;
  /** The group in words, such as season 3 */
  label: string;
  /** merged is a group already queued or searching with this scope */
  state: GrabsQueuedGroupState;
}
