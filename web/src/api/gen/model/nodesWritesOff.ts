import type { NodesHeld } from "./nodesHeld.ts";
import type { NodesNamed } from "./nodesNamed.ts";
import type { NodesWritesOffKind } from "./nodesWritesOffKind.ts";
import type { NodesWritesOffRole } from "./nodesWritesOffRole.ts";

export interface NodesWritesOff {
  held: NodesHeld;
  kind: NodesWritesOffKind;
  node: NodesNamed;
  role: NodesWritesOffRole;
  words: string;
}
