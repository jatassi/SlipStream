import type { NodesBoardNode } from "./nodesBoardNode.ts";
import type { NodesGesture } from "./nodesGesture.ts";
import type { NodesRoleGroup } from "./nodesRoleGroup.ts";

export interface NodesRoleBoard {
  basis: string;
  /** The plan's verdict on every move and add a drag can make */
  gestures: NodesGesture[];
  /** Every Role Group and its Roles, in the table's order */
  groups: NodesRoleGroup[];
  /** Every Node, in id order */
  nodes: NodesBoardNode[];
}
