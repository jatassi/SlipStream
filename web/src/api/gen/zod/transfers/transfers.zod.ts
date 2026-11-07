import * as zod from "zod";

import {
  TransfersSettings,
  TransfersSettingsPatch,
  TransfersSpeedList,
  TransfersTransfer,
  TransfersTransferList,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Transfers between Nodes, with each one's state, progress and speed
 */
export const listTransfersQueryInfoHashRegExp = new RegExp("^[0-9a-f]{40}$");

export const ListTransfersQueryParams = zod.object({
  nodeId: zod.uuid().exactOptional().describe("Only Transfers this Node sends or receives"),
  infoHash: zod
    .string()
    .regex(listTransfersQueryInfoHashRegExp)
    .exactOptional()
    .describe("Only Transfers with a file from this torrent, by its v1 info hash in hex"),
});

export const ListTransfersResponse = TransfersTransferList;

/**
 * @summary Get a Node's Transfer settings
 */
export const GetTransferSettingsParams = zod.object({
  nodeId: zod.uuid(),
});

export const GetTransferSettingsResponse = TransfersSettings;

/**
 * @summary Change a Node's Transfer settings, which the Node applies at once
 */
export const SetTransferSettingsParams = zod.object({
  nodeId: zod.uuid(),
});

export const SetTransferSettingsBody = TransfersSettingsPatch;

export const SetTransferSettingsResponse = TransfersSettings;

/**
 * @summary List the recent speed and route of each pair of Nodes a Transfer moved bytes between
 */
export const ListTransferSpeedsResponse = TransfersSpeedList;

/**
 * @summary Get a Transfer with its state, progress, speed and files
 */
export const GetTransferParams = zod.object({
  id: zod.uuid(),
});

export const GetTransferResponse = TransfersTransfer;
