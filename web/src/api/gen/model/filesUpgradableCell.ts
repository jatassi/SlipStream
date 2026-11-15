import type { FilesLibraryFile } from "./filesLibraryFile.ts";
import type { FilesUpgradableCellState } from "./filesUpgradableCellState.ts";

export interface FilesUpgradableCell {
  /**
   * The Releases its wait holds; 0 while it waits for none
   * @minimum 0
   */
  candidates: number;
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  /** The file filling it, short of its Target's cutoff or upgrade-until score */
  file: FilesLibraryFile;
  /** No file fills it, and a library file that would is gone from its root. Only a search an Admin starts searches it */
  gone: boolean;
  itemId: string;
  /** No Grab is in flight for it. A Grab in flight makes it downloading */
  state: FilesUpgradableCellState;
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
