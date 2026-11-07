import type { IndexersQueryOutcome } from "./indexersQueryOutcome.ts";
import type { IndexersQueryUse } from "./indexersQueryUse.ts";

export interface IndexersQuery {
  /**
   * Null while it runs
   * @nullable
   */
  completedAt: string | null;
  id: string;
  /**
   * Null while it runs; lost when its Node never answered
   * @nullable
   */
  outcome: IndexersQueryOutcome;
  /** @minimum 0 */
  releases: number;
  reservedAt: string;
  /**
   * When its Node sent its first request; null before, or when it sent none
   * @nullable
   */
  startedAt: string | null;
  use: IndexersQueryUse;
}
