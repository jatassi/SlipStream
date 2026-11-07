import type { DownloadclientsTrackerCellStatus } from "./downloadclientsTrackerCellStatus.ts";

export interface DownloadclientsTrackerCell {
  /** Empty when its announce URLs are unknown */
  host: string;
  /** The announce URL's failure reason or warning, verbatim */
  message?: string;
  /** When it tries a failing announce URL again */
  retryAt?: string;
  status: DownloadclientsTrackerCellStatus;
}
