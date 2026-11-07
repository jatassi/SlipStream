import * as zod from "zod";

import { FilesCellList, FilesFileList } from "../slipStream.schemas.zod.ts";

/**
 * @summary Get each of a Work's cells with its state and the file filling it, and each Item's status
 */
export const GetWorkCellsParams = zod.object({
  workId: zod.uuid(),
});

export const GetWorkCellsResponse = FilesCellList;

/**
 * @summary List a Work's library files with the cells each fills, and the folder they sit in
 */
export const ListWorkFilesParams = zod.object({
  workId: zod.uuid(),
});

export const ListWorkFilesResponse = FilesFileList;
