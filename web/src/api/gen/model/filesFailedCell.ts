import type { FilesFailedCellState } from "./filesFailedCellState.ts";

export interface FilesFailedCell {
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  itemId: string;
  state: FilesFailedCellState;
  targetId: string;
  targetName: string;
  /** Missing or failed, and its Item monitored and released */
  wanted: boolean;
}
