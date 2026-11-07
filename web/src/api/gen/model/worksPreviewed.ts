import type { WorksPresetCounts } from "./worksPresetCounts.ts";
import type { WorksTree } from "./worksTree.ts";

export interface WorksPreviewed {
  /** Whether the Work is in the library; while it isn't, the ids in work name nothing */
  inLibrary: boolean;
  /** What each monitoring preset would monitor of the Work; null while it is in the library */
  presets: WorksPresetCounts | null;
  work: WorksTree;
}
