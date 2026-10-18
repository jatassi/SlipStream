import type { TargetsUnwantedFiles } from "./targetsUnwantedFiles.ts";

export interface TargetsImpact {
  /**
   * Downloads in flight for cells that go; their files follow the save's choice
   * @minimum 0
   */
  downloading: number;
  /**
   * Cells that become wanted
   * @minimum 0
   */
  newlyWanted: number;
  /**
   * Works that keep their own Edition list through a change to their Profile's
   * @minimum 0
   */
  ownList: number;
  /**
   * Files that still fill a cell and change name
   * @minimum 0
   */
  renamed: number;
  /** Files that fill a cell now and none after: deleted, or kept as Kept files, as the save chooses */
  unwanted: TargetsUnwantedFiles;
  /**
   * Works whose cells, fills or file names change
   * @minimum 0
   */
  works: number;
}
