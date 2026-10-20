import * as zod from "zod";

import {
  MetadataNewKey,
  MetadataProvider,
  MetadataProviderList,
  MetadataSettings,
  MetadataSettingsChange,
} from "../slipStream.schemas.zod.ts";

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
 * @summary Make one cheap call to a provider with the key in use, and answer its status after it
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
