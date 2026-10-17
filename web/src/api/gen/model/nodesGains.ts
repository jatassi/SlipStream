import type { NodesGainsKind } from "./nodesGainsKind.ts";
import type { NodesGainsRole } from "./nodesGainsRole.ts";
import type { NodesNamed } from "./nodesNamed.ts";

export interface NodesGains {
  kind: NodesGainsKind;
  node: NodesNamed;
  role: NodesGainsRole;
  words: string;
}
