import * as zod from "zod";

import {
  CheckRootInputBody,
  MakeFolderInputBody,
  RootsListing,
  RootsMadeFolder,
  RootsNewRoot,
  RootsRoot,
  RootsRootList,
  RootsSuggestions,
  RootsVerdict,
  RootsWorkRoot,
  SetWorkRootInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List a folder on a Storage Node, or its places
 */
export const BrowseFoldersParams = zod.object({
  id: zod.uuid(),
});

export const browseFoldersQueryPathMax = 4096;

export const BrowseFoldersQueryParams = zod.object({
  path: zod
    .string()
    .max(browseFoldersQueryPathMax)
    .exactOptional()
    .describe("An absolute folder in the Node's path style; empty for its places"),
  hidden: zod.boolean().exactOptional().describe("List hidden and system folders too"),
});

export const BrowseFoldersResponse = RootsListing;

/**
 * @summary Make a folder in an existing folder on a Storage Node
 */
export const MakeFolderParams = zod.object({
  id: zod.uuid(),
});

export const MakeFolderBody = MakeFolderInputBody;

export const MakeFolderResponse = RootsMadeFolder;

/**
 * @summary List the folders on a Storage Node that complete a typed path
 */
export const SuggestFoldersParams = zod.object({
  id: zod.uuid(),
});

export const suggestFoldersQueryTextMax = 4096;

export const SuggestFoldersQueryParams = zod.object({
  text: zod
    .string()
    .max(suggestFoldersQueryTextMax)
    .exactOptional()
    .describe("A path as typed, in the Node's path style"),
});

export const SuggestFoldersResponse = RootsSuggestions;

/**
 * @summary List the library roots
 */
export const ListRootsResponse = RootsRootList;

/**
 * @summary Make a folder on a Storage Node a library root of a Content Type
 */
export const CreateRootBody = RootsNewRoot;

export const CreateRootResponse = RootsRoot;

/**
 * @summary Judge a folder on a Storage Node as a library root without saving it
 */
export const CheckRootBody = CheckRootInputBody;

export const CheckRootResponse = RootsVerdict;

/**
 * @summary Make a root where its Content Type's new Works go
 */
export const MakeDefaultRootParams = zod.object({
  rootId: zod.uuid(),
});

export const MakeDefaultRootResponse = zod.void();

/**
 * @summary Move a Work that has no files into another root of its Content Type
 */
export const SetWorkRootParams = zod.object({
  workId: zod.uuid(),
});

export const SetWorkRootBody = SetWorkRootInputBody;

export const SetWorkRootResponse = RootsWorkRoot;
