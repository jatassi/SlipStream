import type { FilesDownloadingCellState } from "./filesDownloadingCellState.ts";
import type { FilesLibraryFile } from "./filesLibraryFile.ts";
import type { FilesStep } from "./filesStep.ts";

export interface FilesDownloadingCell {
  /**
   * How many Releases it is waiting on; 0 while it waits for none
   * @minimum 0
   */
  candidates: number;
  /** any under any cut, else the key of the cut this Item needs at this Target */
  edition: string;
  /** The file the Grab would replace; null when it replaces none */
  file: FilesLibraryFile | null;
  /**
   * When the Grab entered its stage; null with step
   * @nullable
   */
  from: string | null;
  /** It has no file because its library file is gone from its root. Only a search an Admin starts searches for it */
  gone: boolean;
  /** The latest Grab in flight for it */
  grabId: string;
  itemId: string;
  /** A Grab is in flight for it, fetching included */
  state: FilesDownloadingCellState;
  /** What the Grab or its Import is doing for this Item at this Target; null until SlipStream reads it */
  step: FilesStep | null;
  /** Failed Grabs stopped SlipStream searching it again at once. Backlog search still searches it */
  stopped: boolean;
  targetId: string;
  targetName: string;
  /**
   * When the Import delay ends, for a download that waits for it; null otherwise
   * @nullable
   */
  until: string | null;
  /**
   * When SlipStream grabs the best Release found for it, unless a better one arrives; null while it waits for none
   * @nullable
   */
  waitUntil: string | null;
  /** Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released */
  wanted: boolean;
}
