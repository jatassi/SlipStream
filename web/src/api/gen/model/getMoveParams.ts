export type GetMoveParams = {
  /**
   * The last unit's position on the page before; -1 for the first page
   * @minimum -1
   */
  after?: number;
  /**
   * @minimum 1
   * @maximum 500
   */
  limit?: number;
};
