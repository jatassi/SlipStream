export interface GrabsFetchWait {
  /** Why the fetch waits, such as Waiting to fetch the .torrent from TorrentLeech. SlipStream retries at 14:20. */
  sentence: string;
  /**
   * When SlipStream retries; null while it waits for the Indexer's home Node
   * @nullable
   */
  until: string | null;
}
