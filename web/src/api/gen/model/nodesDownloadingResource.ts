import type { NodesTorrentView } from "./nodesTorrentView.ts";

export interface NodesDownloadingResource {
  /** Each Torrent the client keeps, in added order */
  torrents: NodesTorrentView[];
}
