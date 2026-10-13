import type { FilesLibraryFile } from "./filesLibraryFile.ts";
import type { FilesMetCellEdition } from "./filesMetCellEdition.ts";
import type { FilesMetCellState } from "./filesMetCellState.ts";

export interface FilesMetCell {
  edition: FilesMetCellEdition;
  /** The file filling it */
  file: FilesLibraryFile;
  itemId: string;
  state: FilesMetCellState;
  targetId: string;
  targetName: string;
  /** Missing or failed, and its Item monitored */
  wanted: boolean;
}
