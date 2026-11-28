import type { NodesHeld } from "./nodesHeld.ts";
import type { NodesNamed } from "./nodesNamed.ts";
import type { NodesRequeuesKind } from "./nodesRequeuesKind.ts";
import type { NodesRequeuesRole } from "./nodesRequeuesRole.ts";

export interface NodesRequeues {
  from: NodesNamed;
  held: NodesHeld;
  kind: NodesRequeuesKind;
  role: NodesRequeuesRole;
  words: string;
}
