import type { GrabsFailureCode } from "./grabsFailureCode.ts";
import type { GrabsFailureStage } from "./grabsFailureStage.ts";

export interface GrabsFailure {
  code: GrabsFailureCode;
  /** No later Grab holds any of its cells. Needs you lists a failure only while it is current. */
  current: boolean;
  message: string;
  /** It goes under Needs you. False for a cancel and for a Manual Grab whose Release the Blocklist held after its fetch, which its sheet already said. */
  needsYou: boolean;
  stage: GrabsFailureStage;
}
