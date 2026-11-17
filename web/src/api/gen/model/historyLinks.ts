export interface HistoryLinks {
  /**
   * The Grab it is about; null for none, and once that Grab is deleted
   * @nullable
   */
  grab: string | null;
  /**
   * The key, <category>.<key>, of a one-item Health entry's item while it is on Health; null otherwise
   * @nullable
   */
  health: string | null;
  /**
   * The Import it is about; null for none, and once that Import is deleted
   * @nullable
   */
  import: string | null;
}
