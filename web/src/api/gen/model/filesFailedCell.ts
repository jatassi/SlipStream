import type { FilesFailedCellState } from "./filesFailedCellState.ts";
import type { FilesLibraryFile } from "./filesLibraryFile.ts";

export interface FilesFailedCell {
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  /** The file the Grab would have replaced; null when it fills no file */
  file: FilesLibraryFile | null;
  /** The Grab that failed */
  grabId: string;
  itemId: string;
  state: FilesFailedCellState;
  targetId: string;
  targetName: string;
  /** Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released */
  wanted: boolean;
}
