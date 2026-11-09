export interface RootsStopped {
  /**
   * The recorded files the scan didn't find
   * @minimum 1
   */
  gone: number;
  /**
   * The files the root records
   * @minimum 1
   */
  of: number;
}
