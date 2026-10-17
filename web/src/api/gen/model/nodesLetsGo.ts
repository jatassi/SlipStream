import type { NodesLetsGoKind } from "./nodesLetsGoKind.ts";
import type { NodesLetsGoRole } from "./nodesLetsGoRole.ts";
import type { NodesNamed } from "./nodesNamed.ts";

export interface NodesLetsGo {
  kind: NodesLetsGoKind;
  node: NodesNamed;
  role: NodesLetsGoRole;
  words: string;
}
