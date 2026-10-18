import type { NodesTorrentTraffic } from "./nodesTorrentTraffic.ts";

export interface NodesNodeTraffic {
  /** Payload bytes a second its torrents download */
  downBps: number;
  id: string;
  /** Each Grab whose torrent its client keeps */
  torrents: NodesTorrentTraffic[];
  /** Payload bytes a second its torrents upload */
  upBps: number;
}
