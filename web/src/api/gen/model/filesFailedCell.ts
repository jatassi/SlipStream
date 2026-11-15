import type { FilesFailedCellState } from "./filesFailedCellState.ts";
import type { FilesLibraryFile } from "./filesLibraryFile.ts";

export interface FilesFailedCell {
  /**
   * The Releases its wait holds; 0 while it waits for none
   * @minimum 0
   */
  candidates: number;
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  /** The file the Grab would have replaced; null when it fills no file */
  file: FilesLibraryFile | null;
  /** No file fills it, and a library file that would is gone from its root. Only a search an Admin starts searches it */
  gone: boolean;
  /** The Grab that failed */
  grabId: string;
  itemId: string;
  state: FilesFailedCellState;
  /** Failed Grabs stopped SlipStream searching it again at once. Backlog search still searches it */
  stopped: boolean;
  targetId: string;
  targetName: string;
  /**
   * When SlipStream grabs the best Release found for it, unless a better one arrives; null while it waits for none
   * @nullable
   */
  waitUntil: string | null;
  /** Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released */
  wanted: boolean;
}
