import * as zod from "zod";

import {
  ActInputBody,
  DownloadclientsActed,
  DownloadclientsClientEdit,
  DownloadclientsClientsList,
  DownloadclientsExternalClient,
  DownloadclientsNewClient,
  DownloadclientsSettings,
  DownloadclientsSettingsPatch,
  DownloadclientsTestReport,
  DownloadclientsThresholds,
  DownloadclientsThresholdsPatch,
  DownloadclientsTorrentDetail,
  DownloadclientsTorrentList,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List each downloading Node's built-in and external clients, the kinds of external client, and the thresholds for a bad release
 */
export const ListDownloadClientsResponse = DownloadclientsClientsList;

/**
 * @summary Change when a release counts as bad, which every downloading Node applies at once
 */
export const SetDownloadClientThresholdsBody = DownloadclientsThresholdsPatch;

export const SetDownloadClientThresholdsResponse = DownloadclientsThresholds;

/**
 * @summary Get a Node's built-in torrent client settings and where its new Grabs go
 */
export const GetDownloadClientSettingsParams = zod.object({
  nodeId: zod.uuid(),
});

export const GetDownloadClientSettingsResponse = DownloadclientsSettings;

/**
 * @summary Change a Node's built-in torrent client settings and where its new Grabs go, which the Node applies at once
 */
export const SetDownloadClientSettingsParams = zod.object({
  nodeId: zod.uuid(),
});

export const SetDownloadClientSettingsBody = DownloadclientsSettingsPatch;

export const SetDownloadClientSettingsResponse = DownloadclientsSettings;

/**
 * @summary Add an external Download Client once its Test passed on its Node
 */
export const CreateDownloadClientBody = DownloadclientsNewClient;

export const CreateDownloadClientResponse = DownloadclientsExternalClient;

/**
 * @summary Test an external Download Client from its Node without saving it
 */
export const TestDownloadClientDraftBody = DownloadclientsNewClient;

export const TestDownloadClientDraftResponse = DownloadclientsTestReport;

/**
 * @summary Delete an external Download Client, and with stopTracking stop tracking the torrents it holds
 */
export const DeleteDownloadClientParams = zod.object({
  id: zod.uuid(),
});

export const DeleteDownloadClientQueryParams = zod.object({
  stopTracking: zod
    .boolean()
    .exactOptional()
    .describe(
      "Delete a client that holds torrents SlipStream tracks: they stay in the client, and Grabs not yet imported from them fail",
    ),
});

export const DeleteDownloadClientResponse = zod.void();

/**
 * @summary Change an external Download Client, testing it first when the change is to its Node, kind, address, sign-in, daemon, label, folder or Client folders
 */
export const UpdateDownloadClientParams = zod.object({
  id: zod.uuid(),
});

export const UpdateDownloadClientBody = DownloadclientsClientEdit;

export const UpdateDownloadClientResponse = DownloadclientsExternalClient;

/**
 * @summary Test a saved external Download Client with unsaved changes, using its saved secrets where the changes leave them
 */
export const TestDownloadClientParams = zod.object({
  id: zod.uuid(),
});

export const TestDownloadClientBody = DownloadclientsClientEdit;

export const TestDownloadClientResponse = DownloadclientsTestReport;

/**
 * @summary List the Torrents every Node's Download Clients hold, with their state and counts
 */
export const listTorrentsQueryClientRegExp = new RegExp(
  "^(built-in|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$",
);
export const listTorrentsQueryQMax = 200;

export const ListTorrentsQueryParams = zod.object({
  ids: zod
    .array(zod.uuid())
    .nullish()
    .describe("Only these Torrents, such as the ones a torrents.changed named"),
  tab: zod.enum(["downloading", "seeding", "problems", "all"]).exactOptional(),
  nodeId: zod.uuid().exactOptional(),
  contentType: zod.string().exactOptional(),
  client: zod
    .string()
    .regex(listTorrentsQueryClientRegExp)
    .exactOptional()
    .describe(
      "Only one client's Torrents: an external client's id, or built-in for every Node's built-in client",
    ),
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
 * @summary Pause, resume, announce, recheck, mark failed or remove Torrents, and return the result for each
 */
export const ActOnTorrentsBody = ActInputBody;

export const ActOnTorrentsResponse = DownloadclientsActed;

/**
 * @summary Get a Torrent with its announce URLs, files and Grabs, live from its Node while the Node responds
 */
export const GetTorrentParams = zod.object({
  id: zod.uuid(),
});

export const GetTorrentResponse = DownloadclientsTorrentDetail;
