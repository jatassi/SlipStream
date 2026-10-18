export interface NodesTorrentView {
  /** Bytes downloaded */
  done: number;
  grabId: string;
  peers: number;
  title: string;
  /** Bytes the torrent's selected files hold */
  wanted: number;
}
