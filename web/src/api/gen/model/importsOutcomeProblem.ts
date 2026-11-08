export interface ImportsOutcomeProblem {
  /** The code a request for this pending import alone would have been refused with */
  code: string;
  detail?: string;
  title: string;
}
