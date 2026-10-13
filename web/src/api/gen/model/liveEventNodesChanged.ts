import type { LiveEventNodesChangedEvent } from "./liveEventNodesChangedEvent.ts";
import type { NodesChanged } from "./nodesChanged.ts";

export interface LiveEventNodesChanged {
  event: LiveEventNodesChangedEvent;
  data: NodesChanged;
}
