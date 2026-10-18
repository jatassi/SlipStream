import type { NodesLeavesKind } from "./nodesLeavesKind.ts";
import type { NodesNamed } from "./nodesNamed.ts";

export interface NodesLeaves {
  kind: NodesLeavesKind;
  node: NodesNamed;
  words: string;
}
