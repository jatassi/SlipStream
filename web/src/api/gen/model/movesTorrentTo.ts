export interface MovesTorrentTo {
  /** The external Download Client it moves to; absent for the Node's built-in client */
  clientId?: string;
  /** The Node of the Download Client it moves to */
  nodeId: string;
  torrentId: string;
}
