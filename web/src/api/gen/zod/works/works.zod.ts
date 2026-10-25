import * as zod from "zod";

import {
  WorksAdded,
  WorksCandidateList,
  WorksContentType,
  WorksContentTypeList,
  WorksEnabled,
  WorksNewWork,
  WorksPreviewed,
  WorksSettingsChange,
  WorksTree,
  WorksTypeSettings,
  WorksWorkList,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Content Types this install has, in id order, off ones included
 */
export const ListContentTypesResponse = WorksContentTypeList;

/**
 * @summary Search a Content Type's metadata source
 */
export const searchCatalogPathContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const SearchCatalogParams = zod.object({
  contentType: zod.string().regex(searchCatalogPathContentTypeRegExp),
});

export const SearchCatalogQueryParams = zod.object({
  query: zod.string().min(1),
});

export const SearchCatalogResponse = WorksCandidateList;

/**
 * @summary Read a record of a Content Type's metadata source as the Work adding it would make, without adding it or reading ratings
 */
export const previewCatalogRecordPathContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const previewCatalogRecordPathKeyMax = 200;

export const PreviewCatalogRecordParams = zod.object({
  contentType: zod.string().regex(previewCatalogRecordPathContentTypeRegExp),
  key: zod
    .string()
    .min(1)
    .max(previewCatalogRecordPathKeyMax)
    .describe("The Work's key at its metadata source"),
});

export const PreviewCatalogRecordResponse = WorksPreviewed;

/**
 * @summary Turn a Content Type on or off. It can be off only while it holds no Works
 */
export const setContentTypeEnabledPathContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const SetContentTypeEnabledParams = zod.object({
  contentType: zod.string().regex(setContentTypeEnabledPathContentTypeRegExp),
});

export const SetContentTypeEnabledBody = WorksEnabled;

export const SetContentTypeEnabledResponse = WorksContentType;

/**
 * @summary Get a Content Type's own settings
 */
export const getContentTypeSettingsPathContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const GetContentTypeSettingsParams = zod.object({
  contentType: zod.string().regex(getContentTypeSettingsPathContentTypeRegExp),
});

export const GetContentTypeSettingsResponse = WorksTypeSettings;

/**
 * @summary Change a Content Type's own settings by key, all or none. Every Work of the type may then read differently
 */
export const updateContentTypeSettingsPathContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const UpdateContentTypeSettingsParams = zod.object({
  contentType: zod.string().regex(updateContentTypeSettingsPathContentTypeRegExp),
});

export const UpdateContentTypeSettingsBody = WorksSettingsChange;

export const UpdateContentTypeSettingsResponse = WorksTypeSettings;

/**
 * @summary List the Works, or one Content Type's
 */
export const listWorksQueryContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const ListWorksQueryParams = zod.object({
  contentType: zod.string().regex(listWorksQueryContentTypeRegExp).exactOptional(),
});

export const ListWorksResponse = WorksWorkList;

/**
 * @summary Add a Work by its key with its whole tree, or get the Work already added
 */
export const AddWorkBody = WorksNewWork;

export const AddWorkResponse = WorksAdded;

/**
 * @summary Get a Work's tree with each node's fields
 */
export const GetWorkParams = zod.object({
  workId: zod.uuid(),
});

export const GetWorkResponse = WorksTree;

/**
 * @summary Refresh a Work from its metadata source, keeping every id
 */
export const RefreshWorkParams = zod.object({
  workId: zod.uuid(),
});

export const RefreshWorkResponse = WorksTree;
