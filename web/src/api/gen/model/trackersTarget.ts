export interface TrackersTarget {
  /** The external client; absent for the Node's built-in client */
  clientId?: string;
  nodeId: string;
}
