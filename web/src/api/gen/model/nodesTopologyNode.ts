import type { NodesResources } from "./nodesResources.ts";
import type { NodesRoleView } from "./nodesRoleView.ts";
import type { NodesTopologyNodeState } from "./nodesTopologyNodeState.ts";

export interface NodesTopologyNode {
  controller: boolean;
  id: string;
  /** Its last Sync frame; absent while it is online */
  lastSeenAt?: string;
  name: string;
  resources: NodesResources;
  /** Every Role the Node stands in, in table order */
  roles: NodesRoleView[];
  /** The Node answering, which is the Controller */
  self: boolean;
  /** When it came in step; while away, its last Sync frame */
  since: string;
  state: NodesTopologyNodeState;
}
