import type { TargetsMadeImpact } from "./targetsMadeImpact.ts";
import type { TargetsRemakeImpact } from "./targetsRemakeImpact.ts";
import type { TargetsSubtitles } from "./targetsSubtitles.ts";
import type { TargetsUnwantedFiles } from "./targetsUnwantedFiles.ts";
import type { TargetsWorkRenames } from "./targetsWorkRenames.ts";

export interface TargetsImpact {
  /**
   * Downloads in progress that are no longer needed. Each waits in Pending imports once it finishes
   * @minimum 0
   */
  downloading: number;
  /** The files a change of a Profile's made-from Targets has SlipStream make */
  made: TargetsMadeImpact;
  /**
   * How many more files SlipStream will look for
   * @minimum 0
   */
  newlyWanted: number;
  /**
   * Works that keep their own Edition list through a change to their Profile's
   * @minimum 0
   */
  ownList: number;
  /** The made files a Target's older Conversion settings made */
  remake: TargetsRemakeImpact;
  /**
   * Files whose path changes, Kept files included
   * @minimum 0
   */
  renamed: number;
  /** The first 500 of the renamed files, by Work in title order */
  renames: TargetsWorkRenames[];
  /** What a change of a Profile's subtitle languages does */
  subtitles: TargetsSubtitles;
  /** Files a Target uses now and none uses after: deleted, or kept as Kept files, as the save chooses */
  unwanted: TargetsUnwantedFiles;
  /**
   * Works whose Targets, files or file names change
   * @minimum 0
   */
  works: number;
}
