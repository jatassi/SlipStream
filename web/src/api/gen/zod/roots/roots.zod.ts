import * as zod from "zod";

import {
  RootsNewRoot,
  RootsRoot,
  RootsRootList,
  RootsWorkRoot,
  SetWorkRootInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the library roots
 */
export const ListRootsResponse = RootsRootList;

/**
 * @summary Make a folder on the Controller a library root of a Content Type
 */
export const CreateRootBody = RootsNewRoot;

export const CreateRootResponse = RootsRoot;

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
