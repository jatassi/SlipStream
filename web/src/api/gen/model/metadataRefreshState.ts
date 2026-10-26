export interface MetadataRefreshState {
  /**
   * When every Work of the type was last asked to refresh, by Refresh all or a new language or region; null if never. The first Metadata refresh run started at or after it refreshes them all
   * @nullable
   */
  requestedAt: string | null;
  /** Whether that run is open now */
  running: boolean;
}
