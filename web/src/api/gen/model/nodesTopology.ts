import type { NodesTopologyLink } from "./nodesTopologyLink.ts";
import type { NodesTopologyNode } from "./nodesTopologyNode.ts";

export interface NodesTopology {
  at: string;
  /** The link between the Controller and each Worker, in the Workers' id order */
  links: NodesTopologyLink[];
  /** Every Node, in id order */
  nodes: NodesTopologyNode[];
  /** One sentence naming the Nodes that are offline, reconnecting, finishing their work or covered for, or saying every Node is online */
  sentence: string;
}
