export interface IndexersChallengeSolver {
  /**
   * The address of the service that clears browser challenges for the Node; null when it has none
   * @nullable
   */
  address: string | null;
  /**
   * Seconds the solver may take to clear a page
   * @minimum 10
   * @maximum 180
   */
  timeout: number;
}
