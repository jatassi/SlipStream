import type { WorksOptionsChangeType } from "./worksOptionsChangeType.ts";

export interface WorksOptionsChange {
  /** Whether the Work's files go in a folder per level; left out, it keeps its own */
  levelFolders?: boolean;
  /** Answer what the change would withdraw and write nothing */
  preview: boolean;
  /** The Content Type's own options to change, by key */
  type?: WorksOptionsChangeType;
}
