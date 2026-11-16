import type { GrabsWordedReason } from "./grabsWordedReason.ts";

export interface FormatsTriedTarget {
  name: string;
  /** The name's own refusals, then the reasons at the first Target an automatic search would grab it for, else at the first Target it was judged at */
  reasons: GrabsWordedReason[];
  score: number;
  targetId: string;
  /** An automatic search would grab it for this Target */
  wouldFill: boolean;
}
