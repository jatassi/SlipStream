import type { WorksDeletePlan } from "./worksDeletePlan.ts";
import type { WorksOutcome } from "./worksOutcome.ts";

export interface WorksDeleted {
  /** One per Work, in request order */
  outcomes: WorksOutcome[];
  plan: WorksDeletePlan;
}
