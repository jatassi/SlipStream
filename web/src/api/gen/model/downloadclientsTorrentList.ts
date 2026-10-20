import type { DownloadclientsCounts } from "./downloadclientsCounts.ts";
import type { DownloadclientsTorrentRow } from "./downloadclientsTorrentRow.ts";

export interface DownloadclientsTorrentList {
  /** Every Torrent live or Removing, whatever the filter */
  counts: DownloadclientsCounts;
  /** In the order they were added */
  rows: DownloadclientsTorrentRow[];
}
