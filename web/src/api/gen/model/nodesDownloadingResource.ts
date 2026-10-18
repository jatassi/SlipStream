import type { NodesTorrentView } from "./nodesTorrentView.ts";

export interface NodesDownloadingResource {
  /** Each Grab whose torrent the client keeps */
  torrents: NodesTorrentView[];
}
