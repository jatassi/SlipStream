import type { ImportsFileHoldCode } from "./importsFileHoldCode.ts";

export interface ImportsFileHold {
  /**
   * The tries that ran into the hold
   * @minimum 0
   */
  attempts: number;
  code: ImportsFileHoldCode;
  /** What it waits on and when SlipStream retries */
  sentence: string;
}
