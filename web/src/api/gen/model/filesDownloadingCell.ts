import type { FilesDownloadingCellEdition } from "./filesDownloadingCellEdition.ts";
import type { FilesDownloadingCellState } from "./filesDownloadingCellState.ts";

export interface FilesDownloadingCell {
  edition: FilesDownloadingCellEdition;
  itemId: string;
  state: FilesDownloadingCellState;
  targetId: string;
  targetName: string;
  /** Missing or failed, and its Item monitored and released */
  wanted: boolean;
}
