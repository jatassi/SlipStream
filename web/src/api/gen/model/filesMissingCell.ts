import type { FilesMissingCellState } from "./filesMissingCellState.ts";

export interface FilesMissingCell {
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  itemId: string;
  state: FilesMissingCellState;
  targetId: string;
  targetName: string;
  /** Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released */
  wanted: boolean;
}
