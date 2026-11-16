import type { FilesFailedCellState } from "./filesFailedCellState.ts";
import type { FilesLibraryFile } from "./filesLibraryFile.ts";

export interface FilesFailedCell {
  /**
   * How many Releases it is waiting on; 0 while it waits for none
   * @minimum 0
   */
  candidates: number;
  /** any under any cut, else the key of the cut this Item needs at this Target */
  edition: string;
  /** The file the Grab would have replaced; null when it replaced none */
  file: FilesLibraryFile | null;
  /** It has no file because its library file is gone from its root. Only a search an Admin starts searches for it */
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
