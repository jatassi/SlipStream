/**
 * Its connection to this Controller. A Node on a new machine that stays offline may still be paired with the old Controller
 */
export type ReconcilingNodeRowLink =
  | "in-step"
  | "reconnecting"
  | "offline"
  | "lapsed"
  | "mismatched";
