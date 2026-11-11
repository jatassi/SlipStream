import type { FilesDownloadingCellState } from "./filesDownloadingCellState.ts";
import type { FilesLibraryFile } from "./filesLibraryFile.ts";

export interface FilesDownloadingCell {
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  /** The file the Grab would replace; null when it fills no file */
  file: FilesLibraryFile | null;
  /** The latest Grab in flight for it */
  grabId: string;
  itemId: string;
  /** A Grab is in flight for it, fetching included */
  state: FilesDownloadingCellState;
  targetId: string;
  targetName: string;
  /** Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released */
  wanted: boolean;
}
