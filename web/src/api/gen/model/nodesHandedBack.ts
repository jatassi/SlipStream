import type { NodesHandedBackKind } from "./nodesHandedBackKind.ts";
import type { NodesHandedBackRole } from "./nodesHandedBackRole.ts";
import type { NodesNamed } from "./nodesNamed.ts";

export interface NodesHandedBack {
  /** Every Node that held the Role before */
  from: NodesNamed[];
  kind: NodesHandedBackKind;
  role: NodesHandedBackRole;
  to: NodesNamed;
  words: string;
}
