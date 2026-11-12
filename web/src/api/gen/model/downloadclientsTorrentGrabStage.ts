/**
 * waiting: a file of its Import waits for the Admin and none is sending or placing
 */
export type DownloadclientsTorrentGrabStage =
  | "downloading"
  | "importing"
  | "waiting"
  | "imported"
  | "failed";
