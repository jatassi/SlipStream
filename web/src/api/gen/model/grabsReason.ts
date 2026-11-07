import type { GrabsReasonAttribute } from "./grabsReasonAttribute.ts";
import type { GrabsReasonCode } from "./grabsReasonCode.ts";

export interface GrabsReason {
  /** The attribute a failed filter rules on */
  attribute?: GrabsReasonAttribute;
  /** Why the release was refused, or a note on it. wrong_item is a release whose title names another Item or no Item. covers_more is a release that covers more than one Item, such as a season pack, and SlipStream doesn't grab those yet. */
  code: GrabsReasonCode;
  /** The Target whose verdict it is, absent for wrong_work, wrong_item, covers_more and cut_not_wanted */
  targetId?: string;
  /** The value a filter failed on, the quality a Target doesn't accept, the cut no open cell wants, or what a wrong_item or covers_more title names, such as S01E02 */
  value?: string;
}
