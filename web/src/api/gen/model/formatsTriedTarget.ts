import type { FormatsTriedReason } from "./formatsTriedReason.ts";

export interface FormatsTriedTarget {
  name: string;
  /** The name's own refusals, then the reasons of the first cell it would fill, else of the first cell it was judged at */
  reasons: FormatsTriedReason[];
  score: number;
  targetId: string;
  /** An automatic search would grab it for this Target */
  wouldFill: boolean;
}
