import type { DownloadclientsTorrentRow } from "./downloadclientsTorrentRow.ts";
import type { GrabsGrab } from "./grabsGrab.ts";
import type { HomeWork } from "./homeWork.ts";

export interface HomeJob {
  grab: GrabsGrab;
  /** Its Torrent as the Torrents view lists it, without queuePosition; null until the .torrent is fetched */
  torrent: DownloadclientsTorrentRow | null;
  work: HomeWork;
}
