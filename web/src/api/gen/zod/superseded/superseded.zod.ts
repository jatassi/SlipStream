import * as zod from "zod";

import { SupersededView, WipeInputBody } from "../slipStream.schemas.zod.ts";

/**
 * @summary Retire this Controller after its library was restored on another machine
 */
export const RetireControllerResponse = zod.void();

/**
 * @summary Read whether a restore on another machine replaced this Controller
 */
export const GetSupersededResponse = SupersededView;

/**
 * @summary Resume this Controller, so no earlier proof of a restore fences it again
 */
export const ResumeSupersededControllerResponse = zod.void();

/**
 * @summary Stop this Controller after a Worker said a restore replaced it
 */
export const StopSupersededControllerResponse = zod.void();

/**
 * @summary Back up and wipe this Controller, then join it to the new Controller as a Worker
 */
export const WipeSupersededControllerBody = WipeInputBody;

export const WipeSupersededControllerResponse = zod.void();
