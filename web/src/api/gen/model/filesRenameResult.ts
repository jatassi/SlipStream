export interface FilesRenameResult {
  /** Works left as they are because their plans differ now; plan them again */
  changed: string[];
  /**
   * Files queued to move to their new paths
   * @minimum 0
   */
  queued: number;
}
