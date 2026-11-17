export interface HistoryWorkSummary {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  id: string;
  label: string;
  /**
   * The artwork route for its poster, without a size; null when it has none
   * @nullable
   */
  posterUrl: string | null;
}
