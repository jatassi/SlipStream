import type { FilesCell } from "./filesCell.ts";
import type { FilesItemStatus } from "./filesItemStatus.ts";

export interface FilesCellList {
  /** By Item in tree order, then Target in Profile order */
  cells: FilesCell[];
  /** Every Item in tree order */
  items: FilesItemStatus[];
}
