import type { NodesTopologyLink } from "./nodesTopologyLink.ts";
import type { NodesTopologyNode } from "./nodesTopologyNode.ts";

export interface NodesTopology {
  at: string;
  /** The link between the Controller and each Worker, in the Workers' id order */
  links: NodesTopologyLink[];
  /** Every Node, in id order */
  nodes: NodesTopologyNode[];
  /** What stands out across the Nodes, in plain words */
  sentence: string;
}
