import type { NodesAssignment } from "./nodesAssignment.ts";

export interface NodesRolesChange {
  /** The revision of the Roles the change was planned against; a newer revision fails the call with roles_stale */
  basis: string;
  /** The Roles of each Node the change names. A Node it doesn't name keeps its own. */
  set: NodesAssignment[];
}
