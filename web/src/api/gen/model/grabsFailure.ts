import type { GrabsFailureCode } from "./grabsFailureCode.ts";
import type { GrabsFailureStage } from "./grabsFailureStage.ts";

export interface GrabsFailure {
  code: GrabsFailureCode;
  /** No later Grab is downloading any of the same files. Needs you lists a failure only while it is current. */
  current: boolean;
  message: string;
  /** True when Needs you lists the failure. False for a cancel, and for a Manual Grab whose fetched torrent was on the Blocklist, because the Manual search sheet already showed that. */
  needsYou: boolean;
  stage: GrabsFailureStage;
}
