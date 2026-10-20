export type DownloadclientsTorrentRowState =
  | "removing"
  | "failed"
  | "tracker-error"
  | "paused"
  | "checking"
  | "queued"
  | "stalled"
  | "downloading"
  | "seeding";
