import type { HistoryGroup } from "./historyGroup.ts";

export interface HistoryKindsAnswer {
  /** In filter order */
  groups: HistoryGroup[];
}
