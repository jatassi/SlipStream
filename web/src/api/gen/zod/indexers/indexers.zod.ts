import * as zod from "zod";

import {
  IndexersIndexer,
  IndexersIndexerList,
  IndexersNewAPIKey,
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

/**
 * @summary Enter an Indexer's API key again, once its home Node has read its caps with it
 */
export const SetIndexerAPIKeyParams = zod.object({
  id: zod.uuid(),
});

export const SetIndexerAPIKeyBody = IndexersNewAPIKey;

export const SetIndexerAPIKeyResponse = IndexersIndexer;
