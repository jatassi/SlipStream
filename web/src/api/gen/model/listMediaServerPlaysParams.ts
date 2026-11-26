export type ListMediaServerPlaysParams = {
  /**
   * A player's key; absent for every player
   * @maxLength 256
   */
  player?: string;
  /**
   * The next member of the previous page; absent for the newest
   */
  cursor?: string;
};
