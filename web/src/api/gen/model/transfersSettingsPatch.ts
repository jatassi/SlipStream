export interface TransfersSettingsPatch {
  /**
   * Bytes a second, 0 for no limit; null resets it to its default
   * @nullable
   */
  sendLimit?: number | null;
}
