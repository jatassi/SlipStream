export interface GrabsSearchScope {
  /** @maxItems 20000 */
  itemIds: string[];
  /**
   * Each level stands for every Item under it
   * @maxItems 20000
   */
  levelIds: string[];
}
