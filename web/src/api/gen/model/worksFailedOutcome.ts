import type { WorksFailedOutcomeOutcome } from "./worksFailedOutcomeOutcome.ts";
import type { WorksOutcomeProblem } from "./worksOutcomeProblem.ts";

export interface WorksFailedOutcome {
  outcome: WorksFailedOutcomeOutcome;
  problem: WorksOutcomeProblem;
  workId: string;
}
