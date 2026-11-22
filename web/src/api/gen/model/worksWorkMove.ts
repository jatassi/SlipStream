import type { WorksWorkMovePhase } from "./worksWorkMovePhase.ts";
import type { WorksWorkMoveWaiting } from "./worksWorkMoveWaiting.ts";

export interface WorksWorkMove {
  batchId: string;
  phase: WorksWorkMovePhase;
  toNodeId: string;
  toNodeName: string;
  unitId: string;
  /** What it waits for; absent while it waits for nothing */
  waiting?: WorksWorkMoveWaiting;
  /** What it does now, such as Copying to den */
  words: string;
}
