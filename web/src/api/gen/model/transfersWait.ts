import type { TransfersWaitKind } from "./transfersWaitKind.ts";

export interface TransfersWait {
  kind: TransfersWaitKind;
  /** The Node it waits for */
  nodeId: string;
}
