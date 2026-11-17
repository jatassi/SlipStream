import * as zod from "zod";

import { CalendarAnswer } from "../slipStream.schemas.zod.ts";

/**
 * @summary Read release dates in a span of days
 */
export const getCalendarQueryContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const GetCalendarQueryParams = zod.object({
  from: zod.iso.date().describe("The first day"),
  to: zod.iso.date().describe("The day after the last day, at most 42 days after from"),
  contentType: zod
    .string()
    .regex(getCalendarQueryContentTypeRegExp)
    .exactOptional()
    .describe("Left out, every Content Type that is on"),
  kinds: zod
    .array(zod.string())
    .nullish()
    .describe(
      "A comma list of the dated kinds' keys GET /content-types lists; left out, every kind",
    ),
  unmonitored: zod
    .boolean()
    .exactOptional()
    .describe("Also list unmonitored Items, each in a row of their own"),
});

export const GetCalendarResponse = CalendarAnswer;
