import type { MovesReasonCheck } from "./movesReasonCheck.ts";
import type { MovesReasonColor } from "./movesReasonColor.ts";

export interface MovesReason {
  check: MovesReasonCheck;
  /** Stable per reason, such as space_error or mapping_missing */
  code: string;
  color: MovesReasonColor;
  words: string;
}
