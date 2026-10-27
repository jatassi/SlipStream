export interface WorksOutcomeProblem {
  /** The code a request for this Work alone would have been refused with */
  code: string;
  detail?: string;
  title: string;
}
