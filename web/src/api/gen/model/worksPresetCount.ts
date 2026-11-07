import type { WorksPresetCountPreset } from "./worksPresetCountPreset.ts";

export interface WorksPresetCount {
  /** Live Items the preset monitors, asides left out */
  monitors: number;
  /** Whether the preset monitors Items a refresh adds */
  monitorsNew: boolean;
  preset: WorksPresetCountPreset;
}
