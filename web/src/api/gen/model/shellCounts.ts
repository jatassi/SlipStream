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
  /**
   * The Imports, of any source, with a file waiting for the Admin, ignored ones aside
   * @minimum 0
   */
  pendingImports: number;
}
