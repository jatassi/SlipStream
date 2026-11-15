import type { FilesStateCounts } from "./filesStateCounts.ts";

export interface FilesLevelRollup {
  /** Its monitored Items at each of their Targets, by state */
  cells: FilesStateCounts;
  /** Its monitored Items by status; an Item with no Target isn't counted */
  itemStatus: FilesStateCounts;
  /**
   * Its monitored Items that its source still lists
   * @minimum 0
   */
  items: number;
  levelId: string;
  /** Its Items that its source no longer lists and that still have a file, at each of their Targets, by state */
  withdrawn: FilesStateCounts;
}
