import type { RestoreSummary } from "./restoreSummary.ts";

export interface RestoreStaged {
  /** Names the staged restore in the confirm step */
  id: string;
  summary: RestoreSummary;
}
