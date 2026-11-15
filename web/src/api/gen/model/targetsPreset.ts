import type { TargetsPresetKey } from "./targetsPresetKey.ts";

export interface TargetsPreset {
  key: TargetsPresetKey;
  name: string;
  /** Every quality, best tier first. To apply the preset to chosen qualities, keep only those qualities in these tiers. For no upgrades, put the cutoff at the bottom tier */
  tiers: (string[] | null)[];
}
