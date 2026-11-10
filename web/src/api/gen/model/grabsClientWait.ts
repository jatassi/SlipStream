import type { GrabsGrabClient } from "./grabsGrabClient.ts";

export interface GrabsClientWait {
  /** The external client; null for the Node's built-in client */
  client: GrabsGrabClient | null;
  /** The client's Node */
  nodeId: string;
  /** The Grab is pinned to this client and the client can't take it: it is off or deleted, or its Node doesn't hold Downloading. False while the client's Node is offline */
  off: boolean;
}
