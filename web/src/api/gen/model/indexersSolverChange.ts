export interface IndexersSolverChange {
  /**
   * An http or https address of a service that speaks FlareSolverr's API; empty removes the solver
   * @maxLength 2048
   */
  address: string;
  /** Seconds from 10 to 180; null or left out is 60 */
  timeout?: number;
}
