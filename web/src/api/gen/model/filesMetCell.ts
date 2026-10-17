import type { FilesLibraryFile } from "./filesLibraryFile.ts";
import type { FilesMetCellState } from "./filesMetCellState.ts";

export interface FilesMetCell {
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  /** The file filling it */
  file: FilesLibraryFile;
  itemId: string;
  state: FilesMetCellState;
  targetId: string;
  targetName: string;
  /** Missing or failed, and its Item monitored and released */
  wanted: boolean;
}
