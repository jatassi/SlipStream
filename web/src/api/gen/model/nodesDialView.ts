import type { NodesDialViewBy } from "./nodesDialViewBy.ts";

export interface NodesDialView {
  /** The side that opened the live connection; absent with none */
  by?: NodesDialViewBy;
  /** The addresses the Controller connects to the Node at; empty when the Node connects to the Controller */
  controllerDials: string[];
}
