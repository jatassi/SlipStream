import type { DownloadclientsAggregate } from "./downloadclientsAggregate.ts";
import type { ShellAlerts } from "./shellAlerts.ts";

export interface ShellCounts {
  /**
   * The Grabs neither imported nor failed
   * @minimum 0
   */
  activity: number;
  alerts: ShellAlerts;
  /** Every download at once; torrents.live keeps it current */
  downloads: DownloadclientsAggregate;
}
