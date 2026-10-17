import type { GrabsReasonAttribute } from "./grabsReasonAttribute.ts";
import type { GrabsReasonCode } from "./grabsReasonCode.ts";

export interface GrabsReason {
  /** The attribute a failed filter rules on */
  attribute?: GrabsReasonAttribute;
  code: GrabsReasonCode;
  /** The Target whose verdict it is, absent for wrong_work */
  targetId?: string;
  /** The value a filter failed on, or the quality a Target doesn't accept */
  value?: string;
}
