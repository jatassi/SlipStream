import type { FilesCellNode } from "./filesCellNode.ts";
import type { FilesConvertingCellStage } from "./filesConvertingCellStage.ts";
import type { FilesConvertingCellState } from "./filesConvertingCellState.ts";
import type { FilesLibraryFile } from "./filesLibraryFile.ts";

export interface FilesConvertingCell {
  /**
   * How many Releases it is waiting on; 0 while it waits for none
   * @minimum 0
   */
  candidates: number;
  conversionId: string;
  /** any under any cut, else the key of the cut this Item needs at this Target */
  edition: string;
  /** The file the made file would replace; null when it replaces none */
  file: FilesLibraryFile | null;
  /** It has no file because its library file is gone from its root. Only a search an Admin starts searches for it */
  gone: boolean;
  itemId: string;
  /** The Node it is placed on; null until it is placed */
  node: FilesCellNode | null;
  stage: FilesConvertingCellStage;
  /** An unfinished Conversion makes its file, and it would otherwise be missing or upgradable */
  state: FilesConvertingCellState;
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
