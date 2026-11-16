import type { NodesGestureRole } from "./nodesGestureRole.ts";
import type { NodesGestureVerdict } from "./nodesGestureVerdict.ts";

export interface NodesGesture {
  /** The Node the Role moves from; absent for an add */
  from?: string;
  /** What happens to a Role this drag returns to the Controller, or that the Controller keeps */
  note?: string;
  role: NodesGestureRole;
  to: string;
  verdict: NodesGestureVerdict;
  word: string;
}
