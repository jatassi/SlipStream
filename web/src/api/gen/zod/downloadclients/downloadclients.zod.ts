import * as zod from "zod";

import {
  ActInputBody,
  DownloadclientsActed,
  DownloadclientsClientsList,
  DownloadclientsSettings,
  DownloadclientsSettingsPatch,
  DownloadclientsThresholds,
  DownloadclientsThresholdsPatch,
  DownloadclientsTorrentDetail,
  DownloadclientsTorrentList,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List each downloading Node's built-in client, and the thresholds for a bad release
 */
export const ListDownloadClientsResponse = DownloadclientsClientsList;

/**
 * @summary Change when a release counts as bad, which every downloading Node applies at once
 */
export const SetDownloadClientThresholdsBody = DownloadclientsThresholdsPatch;

export const SetDownloadClientThresholdsResponse = DownloadclientsThresholds;

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

/**
 * @summary List the Torrents every Node's built-in client holds, with their state and counts
 */
export const listTorrentsQueryQMax = 200;

export const ListTorrentsQueryParams = zod.object({
  ids: zod
    .array(zod.uuid())
    .nullish()
    .describe("Only these Torrents, such as the ones a torrents.changed named"),
  tab: zod.enum(["downloading", "seeding", "problems", "all"]).exactOptional(),
  nodeId: zod.uuid().exactOptional(),
  contentType: zod.string().exactOptional(),
  q: zod
    .string()
    .max(listTorrentsQueryQMax)
    .exactOptional()
    .describe(
      "Matches the release name, the Grabs' release titles and their Works' labels, ignoring case",
    ),
});

export const ListTorrentsResponse = DownloadclientsTorrentList;

/**
 * @summary Pause, resume, announce, recheck, mark failed or remove Torrents, answering for each
 */
export const ActOnTorrentsBody = ActInputBody;

export const ActOnTorrentsResponse = DownloadclientsActed;

/**
 * @summary Get a Torrent with its trackers, files and Grabs, live from its Node while it answers
 */
export const GetTorrentParams = zod.object({
  id: zod.uuid(),
});

export const GetTorrentResponse = DownloadclientsTorrentDetail;
