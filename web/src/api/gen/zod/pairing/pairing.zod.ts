import { PairingJoinBody, PairingWorker } from "../slipStream.schemas.zod.ts";

/**
 * @summary Join this Node to a Controller as a Worker with a Join Code
 */
export const ClaimWorkerBody = PairingJoinBody;

export const ClaimWorkerResponse = PairingWorker;
