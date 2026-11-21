import type { RestoreSummaryNode } from "./restoreSummaryNode.ts";

export interface RestoreSummary {
  /** The name of the Controller the Backup is of */
  controller: string;
  /** The Backup is of another machine. Confirm that its Controller stays off */
  newMachine: boolean;
  /**
   * Every Node the Backup lists, the Controller first
   * @nullable
   */
  nodes: RestoreSummaryNode[] | null;
  takenAt: string;
  /** The SlipStream version that took the Backup */
  version: string;
}
