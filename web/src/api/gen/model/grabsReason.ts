import type { GrabsReasonAttribute } from "./grabsReasonAttribute.ts";
import type { GrabsReasonCode } from "./grabsReasonCode.ts";

export interface GrabsReason {
  /** The attribute a failed filter rules on */
  attribute?: GrabsReasonAttribute;
  code: GrabsReasonCode;
  /** The Target whose verdict it is, absent for wrong_work and cut_not_wanted */
  targetId?: string;
  /** The value a filter failed on, the quality a Target doesn't accept, or the cut no open cell wants */
  value?: string;
}
