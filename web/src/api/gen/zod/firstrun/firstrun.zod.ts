import * as zod from "zod";

import { FirstRunStepInputBody, FirstrunFirstRun } from "../slipStream.schemas.zod.ts";

/**
 * @summary Get first setup's steps
 */
export const GetFirstRunResponse = FirstrunFirstRun;

/**
 * @summary Finish first setup. It never reopens
 */
export const FinishFirstRunResponse = zod.void();

/**
 * @summary Mark a first setup step done or skipped
 */
export const SetFirstRunStepParams = zod.object({
  id: zod.string(),
});

export const SetFirstRunStepBody = FirstRunStepInputBody;

export const SetFirstRunStepResponse = FirstrunFirstRun;

/**
 * @summary Check the Setup code without changing anything
 */
export const CheckSetupCodeResponse = zod.void();
