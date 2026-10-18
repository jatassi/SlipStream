import type { NodesDialViewBy } from "./nodesDialViewBy.ts";

export interface NodesDialView {
  /** The side that dialed the live session; absent with none */
  by?: NodesDialViewBy;
  /** The addresses the Controller dials the Node at; empty when the Node dials the Controller */
  controllerDials: string[];
}
