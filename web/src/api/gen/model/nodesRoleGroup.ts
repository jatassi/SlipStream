import type { NodesRoleEntry } from "./nodesRoleEntry.ts";
import type { NodesRoleGroupGroup } from "./nodesRoleGroupGroup.ts";

export interface NodesRoleGroup {
  group: NodesRoleGroupGroup;
  label: string;
  roles: NodesRoleEntry[];
}
