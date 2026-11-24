import * as zod from "zod";

import {
  MovesBatchList,
  MovesBatchPage,
  MovesCanceledMoves,
  MovesMoveAsk,
  MovesMovePreview,
  MovesMoveSettings,
  MovesMoveSettingsValues,
  MovesMoveUnit,
  MovesStartAsk,
  MovesStartedMoves,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Moves' batches, open ones first
 */
export const listMovesQueryLimitDefault = 50;
export const listMovesQueryLimitMax = 100;

export const ListMovesQueryParams = zod.object({
  after: zod.uuid().exactOptional().describe("The last batch of the page before"),
  limit: zod.int().min(1).max(listMovesQueryLimitMax).default(listMovesQueryLimitDefault),
});

export const ListMovesResponse = MovesBatchList;

/**
 * @summary Start a previewed Move. A Start sent again returns the same batch
 */
export const startMovesQueryConfirmMin = 0;

export const StartMovesQueryParams = zod.object({
  confirm: zod
    .int()
    .min(startMovesQueryConfirmMin)
    .describe("The movable count the preview showed"),
});

export const StartMovesBody = MovesStartAsk;

export const StartMovesResponse = MovesStartedMoves;

/**
 * @summary Preview moving Works to other library roots, or torrents to other Download Clients: a verdict for each, and a token to start it with
 */
export const PreviewMovesBody = MovesMoveAsk;

export const PreviewMovesResponse = MovesMovePreview;

/**
 * @summary Get the Moves settings of Settings > Library roots
 */
export const GetMoveSettingsResponse = MovesMoveSettings;

/**
 * @summary Change the Moves settings by key, all or none
 */
export const PatchMoveSettingsBody = MovesMoveSettingsValues;

export const PatchMoveSettingsResponse = MovesMoveSettings;

/**
 * @summary Cancel a unit of a Move before its switch. It removes what it placed
 */
export const CancelMoveUnitParams = zod.object({
  unitId: zod.uuid(),
});

export const CancelMoveUnitResponse = MovesMoveUnit;

/**
 * @summary Queue a failed unit of a Move again
 */
export const RetryMoveUnitParams = zod.object({
  unitId: zod.uuid(),
});

export const RetryMoveUnitResponse = MovesMoveUnit;

/**
 * @summary Switch a unit waiting for playback or a Media Server without waiting longer
 */
export const SwitchMoveUnitNowParams = zod.object({
  unitId: zod.uuid(),
});

export const SwitchMoveUnitNowResponse = MovesMoveUnit;

/**
 * @summary Get a Move's batch with a page of its units
 */
export const GetMoveParams = zod.object({
  batchId: zod.uuid(),
});

export const getMoveQueryAfterDefault = -1;
export const getMoveQueryAfterMin = -1;

export const getMoveQueryLimitDefault = 100;
export const getMoveQueryLimitMax = 500;

export const GetMoveQueryParams = zod.object({
  after: zod
    .int()
    .min(getMoveQueryAfterMin)
    .default(getMoveQueryAfterDefault)
    .describe("The last unit's position on the page before; -1 for the first page"),
  limit: zod.int().min(1).max(getMoveQueryLimitMax).default(getMoveQueryLimitDefault),
});

export const GetMoveResponse = MovesBatchPage;

/**
 * @summary Preview moving a Move's Works back to the roots they left, and its torrents back to the Download Clients they left
 */
export const PreviewMoveBackParams = zod.object({
  batchId: zod.uuid(),
});

export const PreviewMoveBackResponse = MovesMovePreview;

/**
 * @summary Cancel every unit of a Move that hasn't switched. Each one removes what it placed
 */
export const CancelMovesParams = zod.object({
  batchId: zod.uuid(),
});

export const CancelMovesResponse = MovesCanceledMoves;
