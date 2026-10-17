import type { NodesBoardNodeAssignedItem } from "./nodesBoardNodeAssignedItem.ts";
import type { NodesRoleView } from "./nodesRoleView.ts";

export interface NodesBoardNode {
  /** The Roles ticked on the Node, as a set sends them */
  assigned: NodesBoardNodeAssignedItem[];
  /** Out of step, so its checks can't run */
  away: boolean;
  controller: boolean;
  id: string;
  name: string;
  /** Every Role the Node stands in, in table order */
  roles: NodesRoleView[];
}
