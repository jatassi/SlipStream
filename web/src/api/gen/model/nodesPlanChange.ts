import type { NodesAssignment } from "./nodesAssignment.ts";

export interface NodesPlanChange {
  /** The basis of the board or Outcome the change was planned on */
  basis: string;
  /** The Roles of each Node the change names. A Node it doesn't name keeps its own. */
  set: NodesAssignment[];
  /** Run the reach test of each added Role whose target is set, from its Node; setRoles always runs them */
  test?: boolean;
}
