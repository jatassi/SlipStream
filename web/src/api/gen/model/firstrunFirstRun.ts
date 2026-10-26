import type { FirstrunStepState } from "./firstrunStepState.ts";

export interface FirstrunFirstRun {
  /** False once an Admin finished first setup, which never reopens */
  open: boolean;
  /** The shown steps, in order */
  steps: FirstrunStepState[];
}
