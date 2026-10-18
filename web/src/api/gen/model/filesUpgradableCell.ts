import type { FilesLibraryFile } from "./filesLibraryFile.ts";
import type { FilesUpgradableCellState } from "./filesUpgradableCellState.ts";

export interface FilesUpgradableCell {
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  /** The file filling it, below its Target's cutoff */
  file: FilesLibraryFile;
  itemId: string;
  state: FilesUpgradableCellState;
  targetId: string;
  targetName: string;
  /** Missing or failed, and its Item monitored and released */
  wanted: boolean;
}
