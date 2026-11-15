import type { NodesBoardNodeAssignedItem } from "./nodesBoardNodeAssignedItem.ts";
import type { NodesBoardNodeSuggestedItem } from "./nodesBoardNodeSuggestedItem.ts";
import type { NodesProbe } from "./nodesProbe.ts";
import type { NodesRoleView } from "./nodesRoleView.ts";

export interface NodesBoardNode {
  /** The Roles ticked on the Node, as a set sends them */
  assigned: NodesBoardNodeAssignedItem[];
  /** Not connected, so its checks can't run */
  away: boolean;
  controller: boolean;
  id: string;
  name: string;
  /** The last verdict of each probe the Node ran */
  probes: NodesProbe[];
  /** Every Role the Node has, in the order of the Roles table */
  roles: NodesRoleView[];
  /** What a new Node's Roles step ticks first: each Acquire Role that has no probe or whose probe fits on the Node */
  suggested: NodesBoardNodeSuggestedItem[];
}
