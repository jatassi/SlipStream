export interface MovesPlace {
  /** A torrent's Download Client, such as qBittorrent or the built-in client */
  client?: string;
  nodeId: string;
  nodeName: string;
  /** The library root in words, such as Movies library root on den; empty for a torrent */
  root: string;
  rootId: string;
}
