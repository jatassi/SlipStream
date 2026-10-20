import type { DownloadclientsTorrentGrabStage } from "./downloadclientsTorrentGrabStage.ts";

export interface DownloadclientsTorrentGrab {
  id: string;
  itemId: string;
  release: string;
  stage: DownloadclientsTorrentGrabStage;
  workId: string;
}
