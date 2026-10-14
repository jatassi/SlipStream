export type GrabsFailureCode =
  | "indexer_unavailable"
  | "torrent_unreachable"
  | "torrent_invalid"
  | "infohash_mismatch"
  | "torrent_v2_only"
  | "no_download_client"
  | "download_failed"
  | "no_video_file"
  | "probe_failed"
  | "fills_nothing"
  | "transfer_failed"
  | "import_failed";
