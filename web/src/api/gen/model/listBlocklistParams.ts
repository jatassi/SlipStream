export type ListBlocklistParams = {
  /**
   * How many of the newest to skip
   * @minimum 0
   */
  offset?: number;
  /**
   * Only entries whose title holds this text
   * @maxLength 200
   */
  q?: string;
};
