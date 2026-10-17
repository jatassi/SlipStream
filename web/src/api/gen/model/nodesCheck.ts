import type { NodesCheckReason } from "./nodesCheckReason.ts";
import type { NodesCheckRole } from "./nodesCheckRole.ts";
import type { NodesCheckVerdict } from "./nodesCheckVerdict.ts";
import type { NodesNamed } from "./nodesNamed.ts";

export interface NodesCheck {
  node: NodesNamed;
  reason: NodesCheckReason;
  /** Absent for a check on the whole Node */
  role?: NodesCheckRole;
  verdict: NodesCheckVerdict;
  words: string;
}
