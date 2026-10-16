export interface NodesSettingsPatch {
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  joinCodeLifetime?: number | null;
}
