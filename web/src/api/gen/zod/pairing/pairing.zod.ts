import { PairingJoinBody, PairingWorkerClaim } from "../slipStream.schemas.zod.ts";

/**
 * @summary Get the last join that waited for the Controller to connect: still waiting, joined or failed
 */
export const GetWorkerClaimResponse = PairingWorkerClaim;

/**
 * @summary Join this Node to a Controller as a Worker with a Join Code. Returns 202 while it awaits the Controller's dial
 */
export const ClaimWorkerBody = PairingJoinBody;

export const ClaimWorkerResponse = PairingWorkerClaim;
