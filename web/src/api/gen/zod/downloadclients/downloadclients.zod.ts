import * as zod from "zod";

import {
  DownloadclientsSettings,
  DownloadclientsSettingsPatch,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Get a Node's built-in torrent client settings
 */
export const GetDownloadClientSettingsParams = zod.object({
  nodeId: zod.uuid(),
});

export const GetDownloadClientSettingsResponse = DownloadclientsSettings;

/**
 * @summary Change a Node's built-in torrent client settings, which the Node applies at once
 */
export const SetDownloadClientSettingsParams = zod.object({
  nodeId: zod.uuid(),
});

export const SetDownloadClientSettingsBody = DownloadclientsSettingsPatch;

export const SetDownloadClientSettingsResponse = DownloadclientsSettings;
