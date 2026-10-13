import * as zod from "zod";

import { FilesCellList } from "../slipStream.schemas.zod.ts";

/**
 * @summary Get each of a Work's cells with its state and the file filling it
 */
export const GetWorkCellsParams = zod.object({
  workId: zod.uuid(),
});

export const GetWorkCellsResponse = FilesCellList;
