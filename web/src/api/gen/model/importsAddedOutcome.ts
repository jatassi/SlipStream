import type { ImportsAddedOutcomeOutcome } from "./importsAddedOutcomeOutcome.ts";

export interface ImportsAddedOutcome {
  id: string;
  outcome: ImportsAddedOutcomeOutcome;
  /** The Work its files joined */
  workId: string;
}
