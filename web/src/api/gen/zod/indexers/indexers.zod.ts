import * as zod from "zod";

import {
  IndexersIndexer,
  IndexersIndexerEdit,
  IndexersIndexerList,
  IndexersNewIndexer,
  IndexersQueryList,
  IndexersReport,
} from "../slipStream.schemas.zod.ts";

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
