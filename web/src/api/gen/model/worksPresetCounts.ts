import type { WorksPresetCount } from "./worksPresetCount.ts";

export interface WorksPresetCounts {
  /** Items in aside levels, which Include adds to a preset */
  asides: number;
  /** One per preset, in the order the presets show */
  choices: WorksPresetCount[];
}
