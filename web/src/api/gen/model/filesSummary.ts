import type { FilesState } from "./filesState.ts";
import type { FilesStateCounts } from "./filesStateCounts.ts";

export interface FilesSummary {
  /** Its monitored Items' cells by state */
  cells: FilesStateCounts;
  /** Its monitored Items by status; an Item with no cell isn't counted */
  itemStatus: FilesStateCounts;
  /**
   * Its monitored Items that its source still lists, the only Items the status and counts read
   * @minimum 0
   */
  items: number;
  /** The first status its monitored Items hold, in the glossary's order; null while none holds a cell */
  status: FilesState | null;
}
