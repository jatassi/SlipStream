import type { NodesBoardNode } from "./nodesBoardNode.ts";
import type { NodesGesture } from "./nodesGesture.ts";

export interface NodesRoleBoard {
  basis: string;
  /** The plan's verdict on every move and add a drag can make */
  gestures: NodesGesture[];
  /** Every Node, in id order */
  nodes: NodesBoardNode[];
}
