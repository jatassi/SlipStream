import type { FilesDownloadingCellState } from "./filesDownloadingCellState.ts";

export interface FilesDownloadingCell {
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  itemId: string;
  state: FilesDownloadingCellState;
  targetId: string;
  targetName: string;
  /** Missing or failed, and its Item monitored and released */
  wanted: boolean;
}
