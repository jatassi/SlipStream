import type { GrabsFailureCode } from "./grabsFailureCode.ts";
import type { GrabsFailureStage } from "./grabsFailureStage.ts";

export interface GrabsFailure {
  code: GrabsFailureCode;
  message: string;
  stage: GrabsFailureStage;
}
