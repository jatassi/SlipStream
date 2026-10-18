import type { NodesCodeStateState } from "./nodesCodeStateState.ts";
import type { NodesDialing } from "./nodesDialing.ts";

export interface NodesCodeState {
  /** The Node address the Controller dials for an open code; absent without one */
  dialing?: NodesDialing;
  /** The Node that paired with the code; absent unless used */
  nodeId?: string;
  state: NodesCodeStateState;
}
