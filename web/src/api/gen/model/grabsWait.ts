import type { GrabsWaitKind } from "./grabsWaitKind.ts";

export interface GrabsWait {
  kind: GrabsWaitKind;
  /** The Node it waits for */
  nodeId: string;
}
