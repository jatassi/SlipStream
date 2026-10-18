import type { NodesTopologyLinkState } from "./nodesTopologyLinkState.ts";

export interface NodesTopologyLink {
  /** The Controller */
  a: string;
  /** The Worker */
  b: string;
  /** The Node that dialed the live session; absent while the link is down */
  dialer?: string;
  /** The live session's round trip; absent until its first ping is answered */
  rttMs?: number;
  state: NodesTopologyLinkState;
}
