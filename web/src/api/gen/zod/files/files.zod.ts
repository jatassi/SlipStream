import * as zod from "zod";

import {
  FilesCellList,
  FilesFileList,
  FilesRenamePlan,
  FilesRenameResult,
  FilesWorkRenamePlan,
  ForgetInputBody,
  MovingOutputBody,
  PlanInputBody,
  RenameInputBody,
  UnmatchedOutputBody,
  WorkPlanInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Delete the records of gone files, all or none
 */
export const ForgetFilesBody = ForgetInputBody;

export const ForgetFilesResponse = zod.void();

/**
 * @summary Rename the files of each Work whose plan still has the digest shown, and list the Works left because their plans differ now
 */
export const RenameFilesBody = RenameInputBody;

export const RenameFilesResponse = FilesRenameResult;

/**
 * @summary Count the files the disk queue still moves, in a Work, a Content Type or the library
 */
export const CountMovingQueryParams = zod.object({
  workId: zod.uuid().exactOptional().describe("Count only this Work's files"),
  contentType: zod.string().exactOptional().describe("Count only this Content Type's files"),
});

export const CountMovingResponse = MovingOutputBody;

/**
 * @summary Plan the renames of a Work, a Content Type or the library, each Work's counts and digest, under the stored naming settings or a draft
 */
export const PlanRenamesBody = PlanInputBody;

export const PlanRenamesResponse = FilesRenamePlan;

/**
 * @summary Plan the renames of one Work, each file's row with its companions, under the stored naming settings or a draft
 */
export const PlanWorkRenamesParams = zod.object({
  workId: zod.uuid(),
});

export const PlanWorkRenamesBody = WorkPlanInputBody;

export const PlanWorkRenamesResponse = FilesWorkRenamePlan;

/**
 * @summary Count, per Content Type, the files whose path differs from what the naming templates make
 */
export const CountUnmatchedResponse = UnmatchedOutputBody;

/**
 * @summary Get the state of each of a Work's Items at each Target, with the file there, and each Item's status
 */
export const GetWorkCellsParams = zod.object({
  workId: zod.uuid(),
});

export const GetWorkCellsResponse = FilesCellList;

/**
 * @summary List a Work's library files with the Item and Target each one is for, and the folder they sit in
 */
export const ListWorkFilesParams = zod.object({
  workId: zod.uuid(),
});

export const ListWorkFilesResponse = FilesFileList;
