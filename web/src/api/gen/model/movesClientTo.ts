import type { MovesSide } from "./movesSide.ts";

export interface MovesClientTo {
  /** The external Download Client; absent for the Node's built-in client */
  clientId?: string;
  /** The Node of the Download Client whose every torrent moves */
  nodeId: string;
  /** The Download Client they move to */
  to: MovesSide;
}
