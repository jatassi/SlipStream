export interface UnblockInputBody {
  /**
   * @minItems 1
   * @maxItems 1000
   * @items.pattern ^[0-9a-f]{40}$
   */
  infoHashes: string[];
}
