import type { TransfersSpeed } from "./transfersSpeed.ts";

export interface TransfersSpeedList {
  /** One for each source and destination that a Transfer moved bytes between */
  speeds: TransfersSpeed[];
}
