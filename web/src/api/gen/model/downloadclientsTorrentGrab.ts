import type { DownloadclientsTorrentGrabStage } from "./downloadclientsTorrentGrabStage.ts";

export interface DownloadclientsTorrentGrab {
  id: string;
  release: string;
  /** waiting: a file of its Import waits for the Admin and none is sending or placing */
  stage: DownloadclientsTorrentGrabStage;
  workId: string;
}
