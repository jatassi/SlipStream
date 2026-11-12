import type { FilesStateCounts } from "./filesStateCounts.ts";

export interface FilesLevelRollup {
  /** Its monitored Items' cells by state */
  cells: FilesStateCounts;
  /** Its monitored Items by status; an Item with no cell isn't counted */
  itemStatus: FilesStateCounts;
  /**
   * Its monitored Items that its source still lists
   * @minimum 0
   */
  items: number;
  levelId: string;
  /** The cells of its Items its source no longer lists that a file still fills, by state */
  withdrawn: FilesStateCounts;
}
