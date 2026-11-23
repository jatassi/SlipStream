import type { HomeMoveNeedKind } from "./homeMoveNeedKind.ts";
import type { HomeWork } from "./homeWork.ts";

export interface HomeMoveNeed {
  batchId: string;
  kind: HomeMoveNeedKind;
  /** When it began to wait; absent for a failed Move */
  since?: string;
  /** Whether Switch now can end its wait */
  switchNow: boolean;
  unitId: string;
  /** What happened and what to do, such as Heat (1995) has waited 1 hour to switch to den. Someone is playing it on a Media Server. */
  words: string;
  work: HomeWork;
}
