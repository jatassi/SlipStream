import type { NodesLinkTraffic } from "./nodesLinkTraffic.ts";
import type { NodesNodeTraffic } from "./nodesNodeTraffic.ts";

export interface NodesTraffic {
  at: string;
  /** Every live link, between the Controller and a Worker */
  links: NodesLinkTraffic[];
  /** Every Node whose torrent client is sampled, in id order */
  nodes: NodesNodeTraffic[];
}
