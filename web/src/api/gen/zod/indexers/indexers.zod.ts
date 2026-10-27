import * as zod from "zod";

import {
  IndexersCatalogAddress,
  IndexersCatalogEntry,
  IndexersCustomDefinition,
  IndexersDefinitionForm,
  IndexersIndexer,
  IndexersIndexerCatalog,
  IndexersIndexerEdit,
  IndexersIndexerList,
  IndexersNewIndexer,
  IndexersQueryList,
  IndexersReport,
  IndexersSettings,
  IndexersSettingsChange,
  IndexersTracker,
  IndexersTrackerList,
  MergeTrackerInputBody,
  RenameTrackerInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Indexer catalog's definitions and the custom ones, with when the catalog last refreshed
 */
export const GetIndexerCatalogResponse = IndexersIndexerCatalog;

/**
 * @summary Change the Indexer catalog's address once SlipStream reads a schema 12 index there. The next refresh reads it in full.
 */
export const SetCatalogAddressBody = IndexersCatalogAddress;

export const SetCatalogAddressResponse = IndexersIndexerCatalog;

/**
 * @summary Add a custom definition, or replace the custom definition with its id and move that definition's Indexers to it
 */
export const AddCustomDefinitionBody = IndexersCustomDefinition;

export const AddCustomDefinitionResponse = IndexersCatalogEntry;

/**
 * @summary Delete a custom definition that no Indexer runs
 */
export const deleteCustomDefinitionPathIdMax = 200;

export const DeleteCustomDefinitionParams = zod.object({
  id: zod.string().max(deleteCustomDefinitionPathIdMax),
});

export const DeleteCustomDefinitionResponse = zod.void();

/**
 * @summary Get the fields of a definition's add form, from the custom definition with the id or else the catalog's
 */
export const getCatalogDefinitionPathIdMax = 200;

export const GetCatalogDefinitionParams = zod.object({
  id: zod.string().max(getCatalogDefinitionPathIdMax),
});

export const GetCatalogDefinitionResponse = IndexersDefinitionForm;

/**
 * @summary Get the Indexer settings
 */
export const GetIndexerSettingsResponse = IndexersSettings;

/**
 * @summary Change Indexer settings by key, all or none
 */
export const UpdateIndexerSettingsBody = IndexersSettingsChange;

export const UpdateIndexerSettingsResponse = IndexersSettings;

/**
 * @summary List the Indexers
 */
export const ListIndexersResponse = IndexersIndexerList;

/**
 * @summary Add an Indexer once its Test passed on the Node it searches from
 */
export const CreateIndexerBody = IndexersNewIndexer;

export const CreateIndexerResponse = IndexersIndexer;

/**
 * @summary Test an Indexer without saving it
 */
export const TestIndexerDraftBody = IndexersNewIndexer;

export const TestIndexerDraftResponse = IndexersReport;

/**
 * @summary Delete an Indexer with its status, sessions and queries. Its Grabs keep their history.
 */
export const DeleteIndexerParams = zod.object({
  id: zod.uuid(),
});

export const DeleteIndexerResponse = zod.void();

/**
 * @summary Get an Indexer
 */
export const GetIndexerParams = zod.object({
  id: zod.uuid(),
});

export const GetIndexerResponse = IndexersIndexer;

/**
 * @summary Change an Indexer, testing it first when the change is to its home, address or API key
 */
export const UpdateIndexerParams = zod.object({
  id: zod.uuid(),
});

export const UpdateIndexerBody = IndexersIndexerEdit;

export const UpdateIndexerResponse = IndexersIndexer;

/**
 * @summary List an Indexer's searches, fetches and Tests, 50 at a time
 */
export const ListIndexerQueriesParams = zod.object({
  id: zod.uuid(),
});

export const listIndexerQueriesQueryOffsetMin = 0;

export const ListIndexerQueriesQueryParams = zod.object({
  offset: zod
    .int()
    .min(listIndexerQueriesQueryOffsetMin)
    .exactOptional()
    .describe("How many of the newest to skip"),
});

export const ListIndexerQueriesResponse = IndexersQueryList;

/**
 * @summary Test an Indexer from the Node it searches from. A pass clears its backoff and any stop.
 */
export const TestIndexerParams = zod.object({
  id: zod.uuid(),
});

export const TestIndexerResponse = IndexersReport;

/**
 * @summary List the Trackers and the Indexers that reach each
 */
export const ListTrackersResponse = IndexersTrackerList;

/**
 * @summary Get a Tracker
 */
export const GetTrackerParams = zod.object({
  id: zod.uuid(),
});

export const GetTrackerResponse = IndexersTracker;

/**
 * @summary Rename a Tracker. Every Indexer and Release that reached it still does.
 */
export const RenameTrackerParams = zod.object({
  id: zod.uuid(),
});

export const RenameTrackerBody = RenameTrackerInputBody;

export const RenameTrackerResponse = IndexersTracker;

/**
 * @summary Merge a Tracker into another, which keeps its name and takes its Indexers, Grabs, Torrents and hosts. The merged Tracker is deleted.
 */
export const MergeTrackerParams = zod.object({
  id: zod.uuid(),
});

export const MergeTrackerBody = MergeTrackerInputBody;

export const MergeTrackerResponse = IndexersTracker;
