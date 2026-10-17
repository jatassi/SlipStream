import type { FilesUnreleasedCellState } from "./filesUnreleasedCellState.ts";

export interface FilesUnreleasedCell {
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  itemId: string;
  /** Its Item has no release date, or a later one */
  state: FilesUnreleasedCellState;
  targetId: string;
  targetName: string;
  /** Missing or failed, and its Item monitored and released */
  wanted: boolean;
}
