import { SystemHealth, SystemReport, SystemStatus } from "../slipStream.schemas.zod.ts";

/**
 * @summary Report that the Node is serving
 */
export const GetHealthResponse = SystemHealth;

/**
 * @summary Report whether the Node is Unclaimed, a Controller or a Worker, and whether this address can hold a passkey
 */
export const GetStatusResponse = SystemStatus;

/**
 * @summary Report the build, data dir, process and listeners
 */
export const GetSystemResponse = SystemReport;
