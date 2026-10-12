import { SystemHealth, SystemReport, SystemStatus } from "../slipStream.schemas.zod.ts";

/**
 * @summary Report that the Node is serving
 */
export const GetHealthResponse = SystemHealth;

/**
 * @summary Report whether the Node is claimed
 */
export const GetStatusResponse = SystemStatus;

/**
 * @summary Report the build, data dir, process and listeners
 */
export const GetSystemResponse = SystemReport;
