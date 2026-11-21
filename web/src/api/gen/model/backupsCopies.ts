export interface BackupsCopies {
  copied: number;
  copying: number;
  failed: number;
  /** Waiting for a copy to start */
  waiting: number;
}
