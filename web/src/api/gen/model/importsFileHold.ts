import type { ImportsFileHoldCode } from "./importsFileHoldCode.ts";

export interface ImportsFileHold {
  /**
   * How many tries ran into this hold
   * @minimum 0
   */
  attempts: number;
  code: ImportsFileHoldCode;
  /** What it waits on and when SlipStream retries */
  sentence: string;
}
