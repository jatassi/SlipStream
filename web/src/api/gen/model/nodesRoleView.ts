import type { NodesHeld } from "./nodesHeld.ts";
import type { NodesRoleViewRole } from "./nodesRoleViewRole.ts";
import type { NodesRoleViewState } from "./nodesRoleViewState.ts";

export interface NodesRoleView {
  /** The tied work the Node holds for the Role; absent when none */
  held?: NodesHeld;
  role: NodesRoleViewRole;
  state: NodesRoleViewState;
}
