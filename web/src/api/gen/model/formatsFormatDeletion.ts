import type { FormatsTargetRef } from "./formatsTargetRef.ts";
import type { TargetsImpact } from "./targetsImpact.ts";

export interface FormatsFormatDeletion {
  impact: TargetsImpact;
  /** The Targets whose scores for it the delete drops, by name */
  scoredBy: FormatsTargetRef[];
}
