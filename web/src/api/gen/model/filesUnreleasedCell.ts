import type { FilesUnreleasedCellState } from "./filesUnreleasedCellState.ts";

export interface FilesUnreleasedCell {
  /**
   * The Releases its wait holds; 0 while it waits for none
   * @minimum 0
   */
  candidates: number;
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  /** No file fills it, and a library file that would is gone from its root. Only a search an Admin starts searches it */
  gone: boolean;
  itemId: string;
  /** Its Item has no release date, or a later one */
  state: FilesUnreleasedCellState;
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
