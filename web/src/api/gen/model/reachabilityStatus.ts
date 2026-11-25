import type { ReachabilityCounts } from "./reachabilityCounts.ts";
import type { ReachabilityStatusPart } from "./reachabilityStatusPart.ts";

export interface ReachabilityStatus {
  counts: ReachabilityCounts;
  /** The status sentence's parts, joined with · on screen */
  parts: ReachabilityStatusPart[];
}
