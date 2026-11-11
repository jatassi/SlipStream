import type { FilesCell } from "./filesCell.ts";
import type { FilesItemStatus } from "./filesItemStatus.ts";
import type { FilesLevelRollup } from "./filesLevelRollup.ts";
import type { FilesNeed } from "./filesNeed.ts";
import type { FilesSummary } from "./filesSummary.ts";

export interface FilesCellList {
  /** By Item in tree order, then Target in Profile order */
  cells: FilesCell[];
  /** Every Item in tree order */
  items: FilesItemStatus[];
  /** What each level's cells add up to, in the order its first Item comes; empty for a type with no levels */
  levels: FilesLevelRollup[];
  /** What the Work needs from an Admin, failures first */
  needsYou: FilesNeed[];
  /** What the cells add up to for the Work */
  summary: FilesSummary;
}
