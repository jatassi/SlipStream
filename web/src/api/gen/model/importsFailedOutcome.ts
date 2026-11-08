import type { ImportsFailedOutcomeOutcome } from "./importsFailedOutcomeOutcome.ts";
import type { ImportsOutcomeProblem } from "./importsOutcomeProblem.ts";

export interface ImportsFailedOutcome {
  id: string;
  outcome: ImportsFailedOutcomeOutcome;
  problem: ImportsOutcomeProblem;
}
