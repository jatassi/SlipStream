import type { NodesRoleEntryProbe } from "./nodesRoleEntryProbe.ts";
import type { NodesRoleEntryRole } from "./nodesRoleEntryRole.ts";

export interface NodesRoleEntry {
  label: string;
  /** The Controller holds it whatever the Admin ticks */
  pinned: boolean;
  /** The probe a Node runs before it is given the Role; absent for none */
  probe?: NodesRoleEntryProbe;
  role: NodesRoleEntryRole;
}
