import type { NodesAssignment } from "./nodesAssignment.ts";
import type { NodesCheck } from "./nodesCheck.ts";
import type { NodesEffect } from "./nodesEffect.ts";

export interface NodesOutcome {
  /** The basis of the rows it was planned on */
  basis: string;
  checks: NodesCheck[];
  effects: NodesEffect[];
  /** Every Node's written-off Roles once applied */
  forfeit: NodesAssignment[];
  /** Every Node's Roles once applied */
  next: NodesAssignment[];
}
