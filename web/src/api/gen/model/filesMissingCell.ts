import type { FilesMissingCellState } from "./filesMissingCellState.ts";

export interface FilesMissingCell {
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  itemId: string;
  state: FilesMissingCellState;
  targetId: string;
  targetName: string;
  /** Missing or failed, and its Item monitored and released */
  wanted: boolean;
}
