import type { WorksTree } from "./worksTree.ts";

export interface WorksPreviewed {
  /** Whether the Work is in the library; while it isn't, the ids in work name nothing */
  inLibrary: boolean;
  work: WorksTree;
}
