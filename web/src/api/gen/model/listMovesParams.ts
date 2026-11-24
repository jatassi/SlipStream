export type ListMovesParams = {
  /**
   * The last batch of the page before
   */
  after?: string;
  /**
   * @minimum 1
   * @maximum 100
   */
  limit?: number;
};
