import type { NodesBuild } from "./nodesBuild.ts";
import type { NodesLinkState } from "./nodesLinkState.ts";

export interface NodesLink {
  /** Absent while the link is down */
  build?: NodesBuild;
  since: string;
  state: NodesLinkState;
}
