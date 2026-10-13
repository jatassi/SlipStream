import { RootsNewRoot, RootsRoot, RootsRootList } from "../slipStream.schemas.zod.ts";

/**
 * @summary List the library roots
 */
export const ListRootsResponse = RootsRootList;

/**
 * @summary Make a folder on the Controller a Content Type's library root
 */
export const CreateRootBody = RootsNewRoot;

export const CreateRootResponse = RootsRoot;
