import * as zod from "zod";

import {
  MetadataNewKey,
  MetadataProvider,
  MetadataProviderList,
  MetadataRefreshStarted,
  MetadataRefreshState,
  MetadataSettings,
  MetadataSettingsChange,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Get when every Work of a Content Type was last asked to refresh, and whether the run refreshing them is open
 */
export const GetContentTypeRefreshParams = zod.object({
  contentType: zod.string(),
});

export const GetContentTypeRefreshResponse = MetadataRefreshState;

/**
 * @summary Refresh every Work of a Content Type as a run of Metadata refresh now, or at its next run while one is open
 */
export const RefreshContentTypeParams = zod.object({
  contentType: zod.string(),
});

export const RefreshContentTypeResponse = MetadataRefreshStarted;

/**
 * @summary Have every Node reading metadata drop what it cached, the next time it reads
 */
export const ClearMetadataCacheResponse = zod.void();

/**
 * @summary List each metadata provider's status and whose key it is called with
 */
export const ListMetadataProvidersResponse = MetadataProviderList;

/**
 * @summary Remove your key for a provider and go back to SlipStream's
 */
export const ClearMetadataKeyParams = zod.object({
  provider: zod.string(),
});

export const ClearMetadataKeyResponse = MetadataProvider;

/**
 * @summary Use your own key for a provider, once the provider takes it on a test call
 */
export const SetMetadataKeyParams = zod.object({
  provider: zod.string(),
});

export const SetMetadataKeyBody = MetadataNewKey;

export const SetMetadataKeyResponse = MetadataProvider;

/**
 * @summary Make one cheap call to a provider with the key in use, and return the provider's status after it
 */
export const TestMetadataProviderParams = zod.object({
  provider: zod.string(),
});

export const TestMetadataProviderResponse = MetadataProvider;

/**
 * @summary Get the metadata settings, with the language and region options
 */
export const GetMetadataSettingsResponse = MetadataSettings;

/**
 * @summary Change metadata settings by key, all or none
 */
export const UpdateMetadataSettingsBody = MetadataSettingsChange;

export const UpdateMetadataSettingsResponse = MetadataSettings;
