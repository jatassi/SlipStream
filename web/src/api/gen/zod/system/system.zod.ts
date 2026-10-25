import * as zod from "zod";

import {
  SystemHealth,
  SystemNodeConfig,
  SystemReport,
  SystemRestarting,
  SystemStatus,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Report that the Node is serving
 */
export const GetHealthResponse = SystemHealth;

/**
 * @summary Get a Node's settings, by section
 */
export const GetNodeConfigParams = zod.object({
  id: zod.uuid(),
});

export const GetNodeConfigResponse = SystemNodeConfig;

/**
 * @summary Change a Node's settings, every key or none
 */
export const UpdateNodeConfigParams = zod.object({
  id: zod.uuid(),
});

export const UpdateNodeConfigBody = zod
  .record(zod.string(), zod.unknown())
  .describe(
    "Each key's new value: a number, a word, true or false, or null to reset it to its default",
  );

export const UpdateNodeConfigResponse = SystemNodeConfig;

/**
 * @summary Restart a Node. The response comes before it stops
 */
export const RestartNodeParams = zod.object({
  id: zod.uuid(),
});

export const RestartNodeResponse = SystemRestarting;

/**
 * @summary Report a Node's build, data dir, process and listeners
 */
export const GetNodeSystemParams = zod.object({
  id: zod.uuid(),
});

export const GetNodeSystemResponse = SystemReport;

/**
 * @summary Report whether the Node is Unclaimed, a Controller or a Worker, and whether this address can hold a passkey
 */
export const GetStatusResponse = SystemStatus;

/**
 * @summary Report the build, data dir, process and listeners
 */
export const GetSystemResponse = SystemReport;
