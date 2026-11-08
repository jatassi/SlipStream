import type { DownloadclientsTrackerDetailStatus } from "./downloadclientsTrackerDetailStatus.ts";

export interface DownloadclientsTrackerDetail {
  /** The earliest it accepts an announce, set while live */
  acceptsAt?: string;
  /** When it last responded, set while live */
  answeredAt?: string;
  /**
   * The leechers its last announce response counted, set while live and the response carried a count
   * @minimum 0
   */
  leechers?: number;
  /** The announce URL's failure reason or warning, verbatim */
  message?: string;
  /**
   * Its min interval as the announce policy holds to it, set while live
   * @minimum 0
   */
  minIntervalSeconds?: number;
  /** When its next announce is due, set while live */
  nextAnnounce?: string;
  /** When it tries a failing announce URL again, from the last report */
  retryAt?: string;
  /**
   * The seeders its last announce response counted, set while live and the response carried a count
   * @minimum 0
   */
  seeders?: number;
  status: DownloadclientsTrackerDetailStatus;
  /** @minimum 0 */
  tier: number;
  /** With its passkey hidden */
  url: string;
}
