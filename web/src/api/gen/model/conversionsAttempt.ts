import type { ConversionsAttemptOutcome } from "./conversionsAttemptOutcome.ts";

export interface ConversionsAttempt {
  /** The fault, empty when none */
  code: string;
  /**
   * null while the attempt runs
   * @nullable
   */
  endedAt: string | null;
  /** @minimum 1 */
  n: number;
  /** The Node's name */
  node: string;
  /** Absent while the attempt runs */
  outcome?: ConversionsAttemptOutcome;
  startedAt: string;
}
