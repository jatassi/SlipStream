import type { GrabsFailureCode } from "./grabsFailureCode.ts";
import type { GrabsFailureStage } from "./grabsFailureStage.ts";

export interface GrabsFailure {
  /** Why the Grab failed, in words */
  cause: string;
  code: GrabsFailureCode;
  /** No later Grab is downloading any of the same files. Needs you lists a failure only while it is current. */
  current: boolean;
  message: string;
  /** True when Needs you lists the failure. False for a cancel, for a failure SlipStream searches again by itself, and for a Manual Grab whose fetched torrent was on the Blocklist, because the Manual search sheet already showed that. */
  needsYou: boolean;
  /**
   * What to do next, in words; null when there is nothing to do
   * @nullable
   */
  recovery: string | null;
  stage: GrabsFailureStage;
}
