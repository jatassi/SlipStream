import type { NodesTorrentTraffic } from "./nodesTorrentTraffic.ts";

export interface NodesNodeTraffic {
  /** Payload bytes a second its torrents download */
  downBps: number;
  id: string;
  /** Each Torrent its client sampled moving since an Admin began listening, in id order */
  torrents: NodesTorrentTraffic[];
  /** Payload bytes a second its torrents upload */
  upBps: number;
}
