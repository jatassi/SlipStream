import type { GrabsCellKey } from "./grabsCellKey.ts";
import type { GrabsFailureAt } from "./grabsFailureAt.ts";

export interface GrabsStopped {
  cell: GrabsCellKey;
  /** The failed Grabs that stopped it, oldest first */
  failures: GrabsFailureAt[];
  /**
   * When Backlog search searches it next; null while Backlog search is off
   * @nullable
   */
  nextSearchAt: string | null;
}
