import type { FilesCell } from "./filesCell.ts";

export interface FilesCellList {
  /** By Item in tree order, then Target in Profile order */
  cells: FilesCell[];
}
