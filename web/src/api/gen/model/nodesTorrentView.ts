export interface NodesTorrentView {
  /** Bytes downloaded, live while an Admin listens */
  done: number;
  /** 0 while its Node isn't sampled */
  peers: number;
  /** Its newest Grab's release, or its name once no Grab is left */
  title: string;
  torrentId: string;
  /** Bytes the torrent's selected files hold */
  wanted: number;
}
