export interface GrabsIndexerLine {
  /** Automatic search uses it */
  automatic: boolean;
  /** It has an outcome for every Item it was asked about */
  done: boolean;
  /** A fault at the site, as opposed to a limit or no Releases */
  failed: boolean;
  /**
   * Null once the Indexer is deleted
   * @nullable
   */
  indexerId: string | null;
  name: string;
  /** Its outcome in words, such as Limit reached until 14:05, or TorrentLeech searched 6 of 10 episodes. Its limit resets at 14:05. */
  sentence: string;
}
