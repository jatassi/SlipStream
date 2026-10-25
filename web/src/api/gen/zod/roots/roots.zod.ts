import * as zod from "zod";

import {
  CheckFolderInputBody,
  MakeFolderInputBody,
  RehomeRootInputBody,
  RootSettingsInputBody,
  RootsListing,
  RootsMadeFolder,
  RootsNewRoot,
  RootsRoot,
  RootsRootList,
  RootsRootSettings,
  RootsSuggestions,
  RootsVerdict,
  RootsVolumeList,
  RootsWorkRoot,
  SetWorkRootInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List a folder on a Node, or its places
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
 * @summary Make a folder in an existing folder on a Node
 */
export const MakeFolderParams = zod.object({
  id: zod.uuid(),
});

export const MakeFolderBody = MakeFolderInputBody;

export const MakeFolderResponse = RootsMadeFolder;

/**
 * @summary Judge a folder on a Node for a purpose without saving it
 */
export const CheckFolderParams = zod.object({
  id: zod.uuid(),
});

export const CheckFolderBody = CheckFolderInputBody;

export const CheckFolderResponse = RootsVerdict;

/**
 * @summary List the folders on a Node that complete a typed path
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
 * @summary Get the library roots settings
 */
export const GetRootSettingsResponse = RootsRootSettings;

/**
 * @summary Change the library roots settings
 */
export const UpdateRootSettingsBody = RootSettingsInputBody;

export const UpdateRootSettingsResponse = RootsRootSettings;

/**
 * @summary Delete a root that holds no files, leaving its folder on disk
 */
export const DeleteRootParams = zod.object({
  rootId: zod.uuid(),
});

export const DeleteRootQueryParams = zod.object({
  worksTo: zod
    .uuid()
    .exactOptional()
    .describe(
      "The root of the same Content Type its Works go to; required while it has Works and another root of its Content Type exists",
    ),
});

export const DeleteRootResponse = zod.void();

/**
 * @summary Re-home a root onto another folder or Node once every file it records is there
 */
export const RehomeRootParams = zod.object({
  rootId: zod.uuid(),
});

export const RehomeRootBody = RehomeRootInputBody;

export const RehomeRootResponse = RootsRoot;

/**
 * @summary Make a root where its Content Type's new Works go
 */
export const MakeDefaultRootParams = zod.object({
  rootId: zod.uuid(),
});

export const MakeDefaultRootResponse = zod.void();

/**
 * @summary List each Storage Node's volumes that hold a library root
 */
export const ListVolumesResponse = RootsVolumeList;

/**
 * @summary Move a Work that has no files into another root of its Content Type
 */
export const SetWorkRootParams = zod.object({
  workId: zod.uuid(),
});

export const SetWorkRootBody = SetWorkRootInputBody;

export const SetWorkRootResponse = RootsWorkRoot;
