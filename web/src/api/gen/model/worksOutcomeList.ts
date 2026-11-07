import type { WorksOutcome } from "./worksOutcome.ts";

export interface WorksOutcomeList {
  /** One per Work, in request order */
  outcomes: WorksOutcome[];
}
