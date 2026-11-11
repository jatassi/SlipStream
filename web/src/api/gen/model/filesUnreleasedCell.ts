import type { FilesUnreleasedCellState } from "./filesUnreleasedCellState.ts";

export interface FilesUnreleasedCell {
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  itemId: string;
  /** Its Item has no release date, or a later one */
  state: FilesUnreleasedCellState;
  targetId: string;
  targetName: string;
  /** Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released */
  wanted: boolean;
}
