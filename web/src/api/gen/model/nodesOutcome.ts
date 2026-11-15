import type { NodesAssignment } from "./nodesAssignment.ts";
import type { NodesCheck } from "./nodesCheck.ts";
import type { NodesEffect } from "./nodesEffect.ts";

export interface NodesOutcome {
  /** The revision of the Roles it was planned against */
  basis: string;
  checks: NodesCheck[];
  effects: NodesEffect[];
  /** Every Node's Roles whose work is given up once applied */
  forfeit: NodesAssignment[];
  /** Every Node's Roles once applied */
  next: NodesAssignment[];
}
