import type { TargetsImpact } from "./targetsImpact.ts";
import type { WorksOutcome } from "./worksOutcome.ts";

export interface WorksProfileChanged {
  impact: TargetsImpact;
  /** One per Work, in request order */
  outcomes: WorksOutcome[];
}
