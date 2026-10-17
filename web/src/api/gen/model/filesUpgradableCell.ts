import type { FilesLibraryFile } from "./filesLibraryFile.ts";
import type { FilesUpgradableCellEdition } from "./filesUpgradableCellEdition.ts";
import type { FilesUpgradableCellState } from "./filesUpgradableCellState.ts";

export interface FilesUpgradableCell {
  edition: FilesUpgradableCellEdition;
  /** The file filling it, below its Target's cutoff */
  file: FilesLibraryFile;
  itemId: string;
  state: FilesUpgradableCellState;
  targetId: string;
  targetName: string;
  /** Missing or failed, and its Item monitored and released */
  wanted: boolean;
}
