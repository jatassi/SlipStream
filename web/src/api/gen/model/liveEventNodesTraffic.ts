import type { LiveEventNodesTrafficEvent } from "./liveEventNodesTrafficEvent.ts";
import type { NodesTraffic } from "./nodesTraffic.ts";

export interface LiveEventNodesTraffic {
  event: LiveEventNodesTrafficEvent;
  data: NodesTraffic;
}
