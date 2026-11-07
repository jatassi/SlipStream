export interface WorksRowRating {
  /**
   * The value's maximum
   * @exclusiveMinimum 0
   */
  scale: number;
  /** The ratings source, such as imdb */
  source: string;
  /** @minimum 0 */
  value: number;
}
