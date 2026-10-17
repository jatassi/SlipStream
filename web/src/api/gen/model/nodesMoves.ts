import type { NodesHeld } from "./nodesHeld.ts";
import type { NodesMovesKind } from "./nodesMovesKind.ts";
import type { NodesMovesRole } from "./nodesMovesRole.ts";
import type { NodesNamed } from "./nodesNamed.ts";

export interface NodesMoves {
  from: NodesNamed;
  held: NodesHeld;
  kind: NodesMovesKind;
  role: NodesMovesRole;
  to: NodesNamed;
  words: string;
}
