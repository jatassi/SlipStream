import type { NodesWriteOffBodyRolesItem } from "./nodesWriteOffBodyRolesItem.ts";

export interface NodesWriteOffBody {
  /**
   * The Roles whose work on the Node to give up. A Role still ticked on the Node is unticked.
   * @minItems 1
   */
  roles: NodesWriteOffBodyRolesItem[];
}
