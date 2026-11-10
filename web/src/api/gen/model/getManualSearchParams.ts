export type GetManualSearchParams = {
  /**
   * The pins from the last response, with any swap applied. A pin that no longer applies is dropped.
   * @maxItems 2000
   * @nullable
   */
  pins?: string[] | null;
};
