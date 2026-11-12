import type { FilesDownloadingCellState } from "./filesDownloadingCellState.ts";
import type { FilesLibraryFile } from "./filesLibraryFile.ts";
import type { FilesStep } from "./filesStep.ts";

export interface FilesDownloadingCell {
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  /** The file the Grab would replace; null when it fills no file */
  file: FilesLibraryFile | null;
  /**
   * When the Grab entered its stage; null with step
   * @nullable
   */
  from: string | null;
  /** The latest Grab in flight for it */
  grabId: string;
  itemId: string;
  /** A Grab is in flight for it, fetching included */
  state: FilesDownloadingCellState;
  /** What the Grab or its Import is doing for the cell; null until SlipStream reads it */
  step: FilesStep | null;
  targetId: string;
  targetName: string;
  /**
   * When the Import delay ends, for a download that waits for it; null otherwise
   * @nullable
   */
  until: string | null;
  /** Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released */
  wanted: boolean;
}
