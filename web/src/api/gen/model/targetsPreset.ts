import type { TargetsPresetKey } from "./targetsPresetKey.ts";

export interface TargetsPreset {
  key: TargetsPresetKey;
  name: string;
  /** Every quality, best tier first. Over chosen qualities, keep these tiers to them; with no upgrades, put the cutoff at the bottom tier */
  tiers: (string[] | null)[];
}
