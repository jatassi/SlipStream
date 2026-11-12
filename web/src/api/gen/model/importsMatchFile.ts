import type { ImportsMatchFileAction } from "./importsMatchFileAction.ts";

export interface ImportsMatchFile {
  /** Change what the file is */
  action: ImportsMatchFileAction;
  /**
   * The key of the cut it is; null for the one its name carries
   * @nullable
   */
  edition: string | null;
  /**
   * The Items of the Work the file holds, in any order
   * @minItems 1
   */
  itemIds: string[];
  /**
   * The name SlipStream reads its quality and cut from
   * @minLength 1
   */
  releaseName: string;
  workId: string;
}
