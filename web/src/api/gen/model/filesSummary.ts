import type { FilesState } from "./filesState.ts";
import type { FilesStateCounts } from "./filesStateCounts.ts";

export interface FilesSummary {
  /** Its monitored Items at each of their Targets, by state */
  cells: FilesStateCounts;
  /** Its monitored Items by status; an Item with no Target isn't counted */
  itemStatus: FilesStateCounts;
  /**
   * Its monitored Items that its source still lists, the only Items the status and counts read
   * @minimum 0
   */
  items: number;
  /** Its monitored Items' first status in the order failed, missing, downloading, converting, upgradable, unreleased, met; null while none has a Target */
  status: FilesState | null;
}
