/**
 * Why its Node holds it back, as its Node last reported
 */
export type DownloadclientsTorrentRowHold =
  | "wanted"
  | "window"
  | "disk_full"
  | "slot"
  | "check_slot"
  | "disk_guard"
  | "draining"
  | "controller";
