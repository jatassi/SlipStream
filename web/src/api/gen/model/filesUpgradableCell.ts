import type { FilesLibraryFile } from "./filesLibraryFile.ts";
import type { FilesUpgradableCellState } from "./filesUpgradableCellState.ts";

export interface FilesUpgradableCell {
  /** any under any cut, else the key of the cut the cell wants */
  edition: string;
  /** The file filling it, short of its Target's cutoff or upgrade-until score */
  file: FilesLibraryFile;
  itemId: string;
  /** No Grab is in flight for it. A Grab in flight makes it downloading */
  state: FilesUpgradableCellState;
  targetId: string;
  targetName: string;
  /** Missing, failed or upgradable with no Grab in flight, and its Item monitored, listed and released */
  wanted: boolean;
}
