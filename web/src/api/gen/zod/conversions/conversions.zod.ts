import * as zod from "zod";

import {
  CanceledOutputBody,
  ConversionsConversionList,
  ConversionsConversionRow,
  ConversionsGroupPage,
  ConversionsNodeConverting,
  ConversionsNodeSettings,
  MoveInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Conversions tab: its counts, the ones that need attention, the ones converting, and a page of Up next by lane
 */
export const listConversionsQueryQMax = 200;

export const ListConversionsQueryParams = zod.object({
  band: zod
    .enum(["attention", "converting", "next"])
    .exactOptional()
    .describe("Only this band; left out, every band"),
  node: zod.uuid().exactOptional().describe("Only Conversions placed or forecast on this Node"),
  type: zod.string().exactOptional().describe("Only Works of this Content Type"),
  q: zod
    .string()
    .max(listConversionsQueryQMax)
    .exactOptional()
    .describe("Only Works whose title holds this"),
  work: zod.uuid().exactOptional().describe("Only this Work's Conversions"),
  cursor: zod.string().exactOptional().describe("The cursor of the previous page"),
});

export const ListConversionsResponse = ConversionsConversionList;

/**
 * @summary Get a page of a folded group's Conversions
 */
export const GetConversionGroupParams = zod.object({
  groupId: zod.string(),
});

export const GetConversionGroupQueryParams = zod.object({
  cursor: zod.string().exactOptional().describe("The cursor of the previous page"),
});

export const GetConversionGroupResponse = ConversionsGroupPage;

/**
 * @summary Cancel every unfinished Conversion of a folded group
 */
export const CancelConversionGroupParams = zod.object({
  groupId: zod.string(),
});

export const CancelConversionGroupResponse = CanceledOutputBody;

/**
 * @summary Cancel an unfinished Conversion. SlipStream doesn't queue it again until its source file changes or an Admin asks for it
 */
export const CancelConversionParams = zod.object({
  id: zod.uuid(),
});

export const CancelConversionResponse = ConversionsConversionRow;

/**
 * @summary Put a waiting Conversion right after another in its lane
 */
export const MoveConversionParams = zod.object({
  id: zod.uuid(),
});

export const MoveConversionBody = MoveInputBody;

export const MoveConversionResponse = ConversionsConversionRow;

/**
 * @summary Put a waiting Conversion at the head of Moved up
 */
export const MoveConversionUpParams = zod.object({
  id: zod.uuid(),
});

export const MoveConversionUpResponse = ConversionsConversionRow;

/**
 * @summary Queue a failed Conversion again at the head of its lane, with a fresh set of attempts
 */
export const RetryConversionParams = zod.object({
  id: zod.uuid(),
});

export const RetryConversionResponse = ConversionsConversionRow;

/**
 * @summary Get a Node's Converting section: its settings, what it converts with, and its test runs
 */
export const GetNodeConvertingParams = zod.object({
  id: zod.uuid(),
});

export const GetNodeConvertingResponse = ConversionsNodeConverting;

/**
 * @summary Change a Node's Conversions at once, hours and processor switch
 */
export const UpdateNodeConvertingParams = zod.object({
  id: zod.uuid(),
});

export const UpdateNodeConvertingBody = ConversionsNodeSettings;

export const UpdateNodeConvertingResponse = ConversionsNodeConverting;

/**
 * @summary Run a Node's test runs again
 */
export const RerunNodeTestsParams = zod.object({
  id: zod.uuid(),
});

export const RerunNodeTestsResponse = zod.void();
