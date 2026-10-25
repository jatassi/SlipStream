import { ServingWorker } from "../slipStream.schemas.zod.ts";

/**
 * @summary Report this Worker's name, key and Controller, and what it holds and seeds. A Serving Worker responds to this itself
 */
export const GetServingResponse = ServingWorker;
