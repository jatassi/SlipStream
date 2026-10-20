export type DownloadclientsTorrentDetailState =
  | "removing"
  | "failed"
  | "tracker-error"
  | "paused"
  | "checking"
  | "queued"
  | "stalled"
  | "downloading"
  | "seeding";
