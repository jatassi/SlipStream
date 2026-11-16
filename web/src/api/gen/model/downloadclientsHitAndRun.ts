import type { DownloadclientsHitAndRunClears } from "./downloadclientsHitAndRunClears.ts";
import type { DownloadclientsOwes } from "./downloadclientsOwes.ts";
import type { DownloadclientsTrackerRef } from "./downloadclientsTrackerRef.ts";

export interface DownloadclientsHitAndRun {
  /** What meets its Tracker's rule, or none while no rule is known */
  clears: DownloadclientsHitAndRunClears;
  /** What it still needs, absent while its Tracker's rule isn't known */
  owes?: DownloadclientsOwes;
  torrentId: string;
  /** The Tracker it needs more seeding for, null while that Tracker isn't known */
  tracker: DownloadclientsTrackerRef | null;
}
