export interface IndexersCatalogAddress {
  /**
   * The base address, to which SlipStream adds the schema version; null resets it to the default
   * @maxLength 2048
   * @nullable
   */
  address: string | null;
}
