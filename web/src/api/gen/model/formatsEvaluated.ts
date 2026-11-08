export interface FormatsEvaluated {
  kind: string;
  /** After negate */
  matched: boolean;
  name: string;
  negate: boolean;
  required: boolean;
  /** Its pattern ran past the match limit, which counts as not matching */
  timedOut: boolean;
}
