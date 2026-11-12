import type { GrabsWordedReasonAttribute } from "./grabsWordedReasonAttribute.ts";
import type { GrabsWordedReasonCode } from "./grabsWordedReasonCode.ts";

export interface GrabsWordedReason {
  /** The attribute a failed filter rules on */
  attribute?: GrabsWordedReasonAttribute;
  /** Why the release was refused, or a note on it. wrong_item is a release whose title names another Item or no Item. wrong_title is a release whose title or year doesn't match the Work's. blocklisted is a release whose info hash is on the Blocklist. */
  code: GrabsWordedReasonCode;
  /** Whether the reason keeps only an automatic search from grabbing the release. A Manual search can grab past it. False for a hard refusal and for a note. */
  soft: boolean;
  /** The Target whose verdict it is, absent for wrong_work, wrong_item and wrong_title */
  targetId?: string;
  /** The value a filter failed on, the quality a Target doesn't accept, the cut the cell's Edition doesn't take, the score under the minimum, what a wrong_item title names, such as S01E02, or the title and year a wrong_title release gives, such as Heat (1986) */
  value?: string;
  /** The reason as a sentence */
  words: string;
}
