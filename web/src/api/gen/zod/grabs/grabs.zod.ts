import * as zod from "zod";

import {
  GrabsGrab,
  GrabsGrabList,
  GrabsSearchList,
  GrabsSearchStarted,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List every Grab with its stage and progress
 */
export const ListGrabsResponse = GrabsGrabList;

/**
 * @summary Get a Grab with its stage and progress
 */
export const GetGrabParams = zod.object({
  grabId: zod.uuid(),
});

export const GetGrabResponse = GrabsGrab;

/**
 * @summary Search for each of a Work's wanted cells and grab the best release
 */
export const SearchWorkParams = zod.object({
  workId: zod.uuid(),
});

export const SearchWorkResponse = GrabsSearchStarted;

/**
 * @summary List a Work's search runs and their outcomes
 */
export const ListSearchesParams = zod.object({
  workId: zod.uuid(),
});

export const ListSearchesResponse = GrabsSearchList;
