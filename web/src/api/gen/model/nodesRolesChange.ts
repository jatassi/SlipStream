import type { NodesAssignment } from "./nodesAssignment.ts";

export interface NodesRolesChange {
  /** The basis of the board or Outcome the change was planned on */
  basis: string;
  /** The Roles of each Node the change names. A Node it doesn't name keeps its own. */
  set: NodesAssignment[];
}
