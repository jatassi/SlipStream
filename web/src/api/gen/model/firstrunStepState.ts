import type { FirstrunStepID } from "./firstrunStepID.ts";
import type { FirstrunStepStateState } from "./firstrunStepStateState.ts";

export interface FirstrunStepState {
  id: FirstrunStepID;
  state: FirstrunStepStateState;
}
