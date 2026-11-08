import type { DownloadclientsHitAndRunClears } from "./downloadclientsHitAndRunClears.ts";
import type { DownloadclientsOwes } from "./downloadclientsOwes.ts";
import type { DownloadclientsTrackerRef } from "./downloadclientsTrackerRef.ts";

export interface DownloadclientsHitAndRun {
  /** What meets the rule it owes, or none while no rule is known */
  clears: DownloadclientsHitAndRunClears;
  /** What it still owes, absent while its Tracker's rule isn't known */
  owes?: DownloadclientsOwes;
  torrentId: string;
  /** The Tracker it owes, null while that Tracker isn't known */
  tracker: DownloadclientsTrackerRef | null;
}
