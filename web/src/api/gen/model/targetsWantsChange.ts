import type { TargetsImpact } from "./targetsImpact.ts";
import type { TargetsWants } from "./targetsWants.ts";

export interface TargetsWantsChange {
  impact: TargetsImpact;
  wants: TargetsWants;
}
