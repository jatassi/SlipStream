import type { NodesNamed } from "./nodesNamed.ts";
import type { NodesPinnedKind } from "./nodesPinnedKind.ts";
import type { NodesPinnedRole } from "./nodesPinnedRole.ts";

export interface NodesPinned {
  kind: NodesPinnedKind;
  node: NodesNamed;
  role: NodesPinnedRole;
  words: string;
}
