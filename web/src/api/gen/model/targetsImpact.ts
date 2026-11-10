import type { TargetsUnwantedFiles } from "./targetsUnwantedFiles.ts";
import type { TargetsWorkRenames } from "./targetsWorkRenames.ts";

export interface TargetsImpact {
  /**
   * Downloads in progress for cells that are no longer wanted. The save's choice decides whether their files are deleted or kept
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
   * Files whose path changes, Kept files included
   * @minimum 0
   */
  renamed: number;
  /** The first 500 of the renamed files, by Work in title order */
  renames: TargetsWorkRenames[];
  /** Files that fill a cell now and none after: deleted, or kept as Kept files, as the save chooses */
  unwanted: TargetsUnwantedFiles;
  /**
   * Works whose cells, fills or file names change
   * @minimum 0
   */
  works: number;
}
