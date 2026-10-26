import * as zod from "zod";

import {
  WorksAdded,
  WorksCandidateList,
  WorksContentType,
  WorksContentTypeList,
  WorksDeleted,
  WorksDeleting,
  WorksEnabled,
  WorksLibraryRows,
  WorksMonitoredChange,
  WorksNewWork,
  WorksOutcomeList,
  WorksPreviewed,
  WorksProfileChange,
  WorksProfileChanged,
  WorksRefreshing,
  WorksRootChange,
  WorksSearchResult,
  WorksSettingsChange,
  WorksTree,
  WorksTypeSettings,
  WorksWorkList,
  WorksWorkView,
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
 * @summary List every Work as a Library row, or one Content Type's, or the named ones
 */
export const listLibraryQueryContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const listLibraryQueryWorkIdsMax = 200;

export const ListLibraryQueryParams = zod.object({
  contentType: zod
    .string()
    .regex(listLibraryQueryContentTypeRegExp)
    .exactOptional()
    .describe("Only this Content Type's Works"),
  workIds: zod
    .array(zod.uuid())
    .max(listLibraryQueryWorkIdsMax)
    .nullish()
    .describe("Only these Works, such as the ones a works.changed named; not with contentType"),
});

export const ListLibraryResponse = WorksLibraryRows;

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
 * @summary Remove each Work from the library, deleting its files or leaving them, with an outcome per Work
 */
export const DeleteWorksBody = WorksDeleting;

export const DeleteWorksResponse = WorksDeleted;

/**
 * @summary Monitor every Item of each Work, or none, with an outcome per Work
 */
export const SetWorksMonitoredBody = WorksMonitoredChange;

export const SetWorksMonitoredResponse = WorksOutcomeList;

/**
 * @summary Give each Work a Profile through one Impact, with an outcome per Work
 */
export const setWorksProfileQueryUnwantedDefault = `delete`;
export const setWorksProfileQueryConfirmMin = 0;

export const SetWorksProfileQueryParams = zod.object({
  preview: zod.boolean().exactOptional().describe("Return the change's Impact and save nothing"),
  unwanted: zod
    .enum(["delete", "keep"])
    .default(setWorksProfileQueryUnwantedDefault)
    .describe(
      "What becomes of the files the change leaves filling nothing, and of downloads in flight for cells it removes",
    ),
  confirm: zod
    .int()
    .min(setWorksProfileQueryConfirmMin)
    .exactOptional()
    .describe(
      "The unwanted file count the Admin confirmed from the preview: a save that would delete more is refused with impact_changed",
    ),
});

export const SetWorksProfileBody = WorksProfileChange;

export const SetWorksProfileResponse = WorksProfileChanged;

/**
 * @summary Refresh each Work from its metadata source, keeping every id, with an outcome per Work
 */
export const RefreshWorksBody = WorksRefreshing;

export const RefreshWorksResponse = WorksOutcomeList;

/**
 * @summary Move each Work without files to a root, with an outcome per Work
 */
export const SetWorksRootBody = WorksRootChange;

export const SetWorksRootResponse = WorksOutcomeList;

/**
 * @summary Search the library's Works by every title they go by, best first
 */
export const searchWorksQueryQMax = 200;

export const searchWorksQueryLimitDefault = 8;
export const searchWorksQueryLimitMin = 0;
export const searchWorksQueryLimitMax = 20000;

export const searchWorksQueryContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const SearchWorksQueryParams = zod.object({
  q: zod
    .string()
    .min(1)
    .max(searchWorksQueryQMax)
    .describe("Words each match's titles hold as word prefixes, case and accents ignored"),
  limit: zod
    .int()
    .min(searchWorksQueryLimitMin)
    .max(searchWorksQueryLimitMax)
    .default(searchWorksQueryLimitDefault)
    .describe("How many rows to return; 0 returns ids only"),
  contentType: zod
    .string()
    .regex(searchWorksQueryContentTypeRegExp)
    .exactOptional()
    .describe("Only this Content Type's Works"),
});

export const SearchWorksResponse = WorksSearchResult;

/**
 * @summary Get a Work's tree with each node's fields, its links, Profile and root
 */
export const GetWorkParams = zod.object({
  workId: zod.uuid(),
});

export const GetWorkResponse = WorksWorkView;

/**
 * @summary Refresh a Work from its metadata source, keeping every id
 */
export const RefreshWorkParams = zod.object({
  workId: zod.uuid(),
});

export const RefreshWorkResponse = WorksTree;
