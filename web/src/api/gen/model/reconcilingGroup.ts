import type { ReconcilingGroupFound } from "./reconcilingGroupFound.ts";
import type { ReconcilingGroupReason } from "./reconcilingGroupReason.ts";

export interface ReconcilingGroup {
  /** @minimum 0 */
  count: number;
  /** What its Node found of its files */
  found: ReconcilingGroupFound;
  /** not_held: recorded, and no Node reported it after the restore. conflict: two Nodes hold it and both keep it */
  reason: ReconcilingGroupReason;
}
