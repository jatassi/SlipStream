import type { NodesAssignment } from "./nodesAssignment.ts";

export interface NodesPlanChange {
  /** The revision of the Roles the change was planned against; a newer revision fails the call with roles_stale */
  basis: string;
  /** The Roles of each Node the change names. A Node it doesn't name keeps its own. */
  set: NodesAssignment[];
  /** Run the reach test of each added Role whose target is set, from its Node; setRoles always runs them */
  test?: boolean;
}
