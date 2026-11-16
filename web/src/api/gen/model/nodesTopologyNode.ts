import type { NodesProbe } from "./nodesProbe.ts";
import type { NodesResources } from "./nodesResources.ts";
import type { NodesRoleView } from "./nodesRoleView.ts";
import type { NodesTopologyNodeState } from "./nodesTopologyNodeState.ts";

export interface NodesTopologyNode {
  controller: boolean;
  id: string;
  /** Its last Sync frame; absent while it is online */
  lastSeenAt?: string;
  name: string;
  /** The last verdict of each probe the Node ran, kept while it is away */
  probes: NodesProbe[];
  resources: NodesResources;
  /** Every Role the Node has, in the order of the Roles table */
  roles: NodesRoleView[];
  /** The Node that responded, which is the Controller */
  self: boolean;
  /** When it connected; while away, when it last reported */
  since: string;
  state: NodesTopologyNodeState;
  /** Whether any Media Server the Node reaches responds; false unless it has the Media Server link Role */
  television: boolean;
}
