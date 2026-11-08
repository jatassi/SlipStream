export interface NodesHeld {
  /** 0 when sizes don't apply */
  bytes: number;
  /**
   * How many units the Node can't finish on its own, which stay until they are removed, moved or written off
   * @minimum 0
   */
  endlessUnits: number;
  /** The noun that fits units: torrent, torrents */
  noun: string;
  units: number;
}
