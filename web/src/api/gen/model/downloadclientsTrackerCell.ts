import type { DownloadclientsTrackerCellStatus } from "./downloadclientsTrackerCellStatus.ts";

export interface DownloadclientsTrackerCell {
  /** Empty when its tracker list is unknown */
  host: string;
  /** The tracker's failure reason or warning, verbatim */
  message?: string;
  /** When it tries a failing tracker again */
  retryAt?: string;
  status: DownloadclientsTrackerCellStatus;
}
