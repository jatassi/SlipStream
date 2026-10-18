import type { TargetsPreset } from "./targetsPreset.ts";
import type { TargetsQuality } from "./targetsQuality.ts";

export interface TargetsQualityList {
  /** The default first */
  presets: TargetsPreset[];
  /** Low to high, the presets' default order */
  qualities: TargetsQuality[];
}
