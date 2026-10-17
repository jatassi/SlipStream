import type { NodesDrainsKind } from "./nodesDrainsKind.ts";
import type { NodesDrainsRole } from "./nodesDrainsRole.ts";
import type { NodesHeld } from "./nodesHeld.ts";
import type { NodesNamed } from "./nodesNamed.ts";

export interface NodesDrains {
  held: NodesHeld;
  kind: NodesDrainsKind;
  node: NodesNamed;
  role: NodesDrainsRole;
  words: string;
}
