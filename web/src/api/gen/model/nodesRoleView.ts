import type { NodesHeld } from "./nodesHeld.ts";
import type { NodesRoleViewRole } from "./nodesRoleViewRole.ts";
import type { NodesRoleViewState } from "./nodesRoleViewState.ts";

export interface NodesRoleView {
  /** The work the Node has for the Role that can't move to another Node; absent when none */
  held?: NodesHeld;
  role: NodesRoleViewRole;
  state: NodesRoleViewState;
}
