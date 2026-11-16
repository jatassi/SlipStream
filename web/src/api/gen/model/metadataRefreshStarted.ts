export interface MetadataRefreshStarted {
  /**
   * The Metadata refresh run that refreshes every Work of the type; null while a run is already in progress, and the next run refreshes them
   * @nullable
   */
  runId: number | null;
}
