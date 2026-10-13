import type { WorksTree } from "./worksTree.ts";

export interface WorksAdded {
  /** False when the Work was already in the library */
  created: boolean;
  work: WorksTree;
}
