import * as zod from "zod";

import {
  GrabsGrab,
  GrabsGrabList,
  GrabsIndexerGrabList,
  GrabsReleaseList,
  GrabsSearchList,
  GrabsSearchScope,
  GrabsSearchStarted,
  SendToInputBody,
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
 * @summary Send a Grab to another Download Client: a downloading Grab's torrent, or a Grab whose fetch waits for its client's Node, which then fetches for the new client
 */
export const SendGrabToParams = zod.object({
  grabId: zod.uuid(),
});

export const SendGrabToBody = SendToInputBody;

export const SendGrabToResponse = zod.void();

/**
 * @summary List the Grabs taken from an Indexer, 50 at a time
 */
export const ListIndexerGrabsParams = zod.object({
  id: zod.uuid(),
});

export const listIndexerGrabsQueryOffsetMin = 0;

export const ListIndexerGrabsQueryParams = zod.object({
  offset: zod
    .int()
    .min(listIndexerGrabsQueryOffsetMin)
    .exactOptional()
    .describe("How many of the newest to skip"),
});

export const ListIndexerGrabsResponse = GrabsIndexerGrabList;

/**
 * @summary List the Releases a search run saw, with what it did with each and why
 */
export const ListSearchReleasesParams = zod.object({
  searchId: zod.uuid(),
});

export const ListSearchReleasesResponse = GrabsReleaseList;

/**
 * @summary Search for each wanted Item of a Work, or of some of its levels and Items, and grab the best release
 */
export const SearchWorkParams = zod.object({
  workId: zod.uuid(),
});

export const SearchWorkBody = GrabsSearchScope;

export const SearchWorkResponse = GrabsSearchStarted;

/**
 * @summary List a Work's search runs and their outcomes
 */
export const ListSearchesParams = zod.object({
  workId: zod.uuid(),
});

export const ListSearchesResponse = GrabsSearchList;
