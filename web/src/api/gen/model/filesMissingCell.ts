import type { FilesMissingCellEdition } from "./filesMissingCellEdition.ts";
import type { FilesMissingCellState } from "./filesMissingCellState.ts";

export interface FilesMissingCell {
  edition: FilesMissingCellEdition;
  itemId: string;
  state: FilesMissingCellState;
  targetId: string;
  targetName: string;
  /** Missing or failed, and its Item monitored and released */
  wanted: boolean;
}
