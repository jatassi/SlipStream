import {
  IndexersIndexer,
  IndexersIndexerList,
  IndexersNewIndexer,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Indexers
 */
export const ListIndexersResponse = IndexersIndexerList;

/**
 * @summary Add a Torznab Indexer, once its home Node has read its caps
 */
export const CreateIndexerBody = IndexersNewIndexer;

export const CreateIndexerResponse = IndexersIndexer;
