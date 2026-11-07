import type { WorksWorkView } from "./worksWorkView.ts";

export interface WorksAdded {
  /** False when the Work was already in the library */
  created: boolean;
  work: WorksWorkView;
}
