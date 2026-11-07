import type { FilesCell } from "./filesCell.ts";
import type { FilesItemStatus } from "./filesItemStatus.ts";
import type { FilesNeed } from "./filesNeed.ts";
import type { FilesSummary } from "./filesSummary.ts";

export interface FilesCellList {
  /** By Item in tree order, then Target in Profile order */
  cells: FilesCell[];
  /** Every Item in tree order */
  items: FilesItemStatus[];
  /** What the Work needs from an Admin, failures first */
  needsYou: FilesNeed[];
  /** What the cells add up to for the Work */
  summary: FilesSummary;
}
