import type { NodesAssignmentRolesItem } from "./nodesAssignmentRolesItem.ts";

export interface NodesAssignment {
  nodeId: string;
  roles: NodesAssignmentRolesItem[];
}
