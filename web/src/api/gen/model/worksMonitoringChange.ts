import type { WorksMonitoringChangePreset } from "./worksMonitoringChangePreset.ts";

export interface WorksMonitoringChange {
  /** Monitor the aside Items the preset picks, and new ones when it monitors new Items */
  includeAsides: boolean;
  /** Whether to monitor Items a refresh adds; left out, the preset decides */
  monitorNew?: boolean;
  preset: WorksMonitoringChangePreset;
  /** Return what the change would start and stop monitoring, and save nothing */
  preview: boolean;
  /**
   * A repeated id counts once
   * @minItems 1
   * @maxItems 20000
   */
  workIds: string[];
}
