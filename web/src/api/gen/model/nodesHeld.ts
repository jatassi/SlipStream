export interface NodesHeld {
  /** 0 when sizes don't apply */
  bytes: number;
  /** The Node can't finish it on its own, so it stays until it is moved or written off */
  endless: boolean;
  /** The noun that fits units: torrent, torrents */
  noun: string;
  units: number;
}
