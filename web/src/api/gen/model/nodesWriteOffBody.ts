import type { NodesWriteOffBodyRolesItem } from "./nodesWriteOffBodyRolesItem.ts";

export interface NodesWriteOffBody {
  /**
   * The Roles whose work the Node holds to give up. A Role still ticked on the Node is unticked.
   * @minItems 1
   */
  roles: NodesWriteOffBodyRolesItem[];
}
