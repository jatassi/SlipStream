import type { DownloadclientsNodeTorrents } from "./downloadclientsNodeTorrents.ts";

export interface DownloadclientsNodeTorrentsList {
  /** One per Downloading Node, in Node order */
  nodes: DownloadclientsNodeTorrents[];
}
