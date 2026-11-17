export type ListNotificationsParams = {
  /**
   * The before of the previous page; left out for the newest
   * @minimum 0
   */
  before?: number;
  /**
   * @minimum 1
   * @maximum 100
   */
  limit?: number;
};
