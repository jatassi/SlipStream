import type { FilesFailedCellEdition } from "./filesFailedCellEdition.ts";
import type { FilesFailedCellState } from "./filesFailedCellState.ts";

export interface FilesFailedCell {
  edition: FilesFailedCellEdition;
  itemId: string;
  state: FilesFailedCellState;
  targetId: string;
  targetName: string;
  /** Missing or failed, and its Item monitored */
  wanted: boolean;
}
