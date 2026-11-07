export interface IndexersCatalogChanged {
  /**
   * Null until a refresh has read the catalog
   * @nullable
   */
  refreshedAt: string | null;
}
