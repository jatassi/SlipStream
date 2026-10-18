import type { ShellAlerts } from "./shellAlerts.ts";

export interface ShellCounts {
  /**
   * The Grabs neither imported nor failed
   * @minimum 0
   */
  activity: number;
  alerts: ShellAlerts;
}
