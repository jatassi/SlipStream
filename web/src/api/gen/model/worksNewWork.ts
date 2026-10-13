export interface WorksNewWork {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /**
   * The Work's key at its metadata source
   * @minLength 1
   * @maxLength 200
   */
  key: string;
}
