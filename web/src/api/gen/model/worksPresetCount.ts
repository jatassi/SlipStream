import type { WorksPresetCountPreset } from "./worksPresetCountPreset.ts";
import type { WorksScopeCount } from "./worksScopeCount.ts";

export interface WorksPresetCount {
  /** Live Items the preset monitors, asides left out */
  monitors: number;
  /** Whether the preset monitors Items a refresh adds */
  monitorsNew: boolean;
  preset: WorksPresetCountPreset;
  /** How many of the Items the preset monitors each add search scope searches */
  searches: WorksScopeCount[];
}
