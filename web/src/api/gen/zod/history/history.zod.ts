import * as zod from "zod";

import {
  HistoryKindsAnswer,
  HistoryListed,
  HistoryOneEntry,
  HistorySentence,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Delete every History entry
 */
export const ClearHistoryResponse = zod.void();

/**
 * @summary List History entries, newest first
 */

export const listHistoryQueryLimitDefault = 50;
export const listHistoryQueryLimitMax = 100;

export const listHistoryQueryContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const ListHistoryQueryParams = zod.object({
  before: zod
    .int()
    .min(1)
    .exactOptional()
    .describe(
      "Only entries with a smaller id: the next member of the previous page; left out, the newest",
    ),
  limit: zod.int().min(1).max(listHistoryQueryLimitMax).default(listHistoryQueryLimitDefault),
  contentType: zod
    .string()
    .regex(listHistoryQueryContentTypeRegExp)
    .exactOptional()
    .describe(
      "Only entries about this Content Type's Works; left out, every entry, Health included",
    ),
  groups: zod
    .array(zod.enum(["grabs", "imports", "library", "subtitles", "health", "backups", "moves"]))
    .nullish()
    .describe("Only these groups; left out, all seven"),
  kinds: zod
    .array(zod.string())
    .nullish()
    .describe(
      "A comma list of the kinds GET /history/kinds lists. Each kind narrows its own group, and groups, when given, must include that group",
    ),
  since: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("Only entries captured at or after this instant"),
  work: zod.uuid().exactOptional().describe("Only entries about this Work"),
});

export const ListHistoryResponse = HistoryListed;

/**
 * @summary List History's groups and kinds
 */
export const ListHistoryKindsResponse = HistoryKindsAnswer;

/**
 * @summary Count what happened since an instant
 */
export const GetHistorySummaryQueryParams = zod.object({
  since: zod.iso
    .datetime({ offset: true })
    .describe("The instant the week starts: the browser's midnight 6 days back"),
});

export const GetHistorySummaryResponse = zod
  .union([HistorySentence, zod.null()])
  .describe("null when History is empty and no Grab needs attention");

/**
 * @summary Read one History entry in full
 */

export const GetHistoryEntryParams = zod.object({
  id: zod.int().min(1),
});

export const GetHistoryEntryQueryParams = zod.object({
  zone: zod
    .string()
    .describe("The IANA time zone the clock times are worded in, such as Europe/London"),
});

export const GetHistoryEntryResponse = HistoryOneEntry;
