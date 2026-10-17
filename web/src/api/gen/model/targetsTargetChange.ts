import type { TargetsImpact } from "./targetsImpact.ts";
import type { TargetsTarget } from "./targetsTarget.ts";

export interface TargetsTargetChange {
  impact: TargetsImpact;
  target: TargetsTarget;
}
