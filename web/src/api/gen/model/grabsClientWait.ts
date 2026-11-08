import type { GrabsGrabClient } from "./grabsGrabClient.ts";

export interface GrabsClientWait {
  /** The external client; null for the Node's built-in client */
  client: GrabsGrabClient | null;
  /** The client's Node, which is offline */
  nodeId: string;
}
