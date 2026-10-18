import type { TargetsCut } from "./targetsCut.ts";
import type { TargetsImpact } from "./targetsImpact.ts";

export interface TargetsCutChange {
  cut: TargetsCut;
  impact: TargetsImpact;
}
