export interface MetadataRefreshAllRun {
  /**
   * When the latest Refresh all of the type ran; null until one has
   * @nullable
   */
  lastRunAt: string | null;
  /** Whether a run that refreshes every Work of the type is open */
  running: boolean;
}
