import type { FilesHeldCell } from "./filesHeldCell.ts";
import type { FilesLibraryFile } from "./filesLibraryFile.ts";
import type { FilesMetCellState } from "./filesMetCellState.ts";

export interface FilesMetCell {
  /**
   * The Releases its wait holds; 0 while it waits for none
   * @minimum 0
   */
  candidates: number;
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  /** The file filling it */
  file: FilesLibraryFile;
  /** No file fills it, and a library file that would is gone from its root. Only a search an Admin starts searches it */
  gone: boolean;
  /** The Grab in flight for it; null while none is */
  held: FilesHeldCell | null;
  itemId: string;
  state: FilesMetCellState;
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
