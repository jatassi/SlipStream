import type { DownloadclientsAggregate } from "./downloadclientsAggregate.ts";
import type { ShellAlerts } from "./shellAlerts.ts";
import type { ShellBanner } from "./shellBanner.ts";

export interface ShellCounts {
  /**
   * The Grabs neither imported nor failed, and the Conversions running
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
  /** What Reconciling after a restore waits on, for the shell's line; null outside it. reconciling.changed keeps it current */
  reconciling: ShellBanner | null;
}
