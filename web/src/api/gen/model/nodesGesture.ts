import type { NodesGestureRole } from "./nodesGestureRole.ts";
import type { NodesGestureVerdict } from "./nodesGestureVerdict.ts";

export interface NodesGesture {
  /** The Node the Role moves from; absent for an add */
  from?: string;
  /** The words of a Role the gesture hands back or pins */
  note?: string;
  role: NodesGestureRole;
  to: string;
  verdict: NodesGestureVerdict;
  word: string;
}
