import type { FilesLibraryFile } from "./filesLibraryFile.ts";
import type { FilesMetCellState } from "./filesMetCellState.ts";

export interface FilesMetCell {
  /** The file filling it */
  file: FilesLibraryFile;
  state: FilesMetCellState;
}
