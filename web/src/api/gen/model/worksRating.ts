export interface WorksRating {
  fetchedAt: string;
  /** The source's name as a person reads it */
  label: string;
  /**
   * The value's maximum
   * @exclusiveMinimum 0
   */
  scale: number;
  /** The ratings source, such as imdb */
  source: string;
  /** @minimum 0 */
  value: number;
  /**
   * Null when the source counts none
   * @minimum 0
   * @nullable
   */
  votes: number | null;
}
