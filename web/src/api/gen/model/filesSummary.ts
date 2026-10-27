import type { FilesState } from "./filesState.ts";
import type { FilesStateCounts } from "./filesStateCounts.ts";

export interface FilesSummary {
  /** Its cells by state */
  cells: FilesStateCounts;
  /** Its Items by status; an Item with no cell isn't counted */
  itemStatus: FilesStateCounts;
  /** @minimum 0 */
  items: number;
  /** The first status its Items hold, in the glossary's order; null for a Work with no cell */
  status: FilesState | null;
}
