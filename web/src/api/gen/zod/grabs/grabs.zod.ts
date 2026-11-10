import * as zod from "zod";

import {
  CancelInputBody,
  GrabManualInputBody,
  GrabsBlocklist,
  GrabsGrab,
  GrabsGrabList,
  GrabsIndexerGrabList,
  GrabsManualSearch,
  GrabsManualStarted,
  GrabsOfferList,
  GrabsPickResults,
  GrabsReleaseList,
  GrabsSearchList,
  GrabsSearchScope,
  GrabsSearchStarted,
  GrabsStartManual,
  SendToInputBody,
  UnblockInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Blocklist, 50 entries at a time, newest first
 */
export const listBlocklistQueryOffsetMin = 0;

export const listBlocklistQueryQMax = 200;

export const ListBlocklistQueryParams = zod.object({
  offset: zod
    .int()
    .min(listBlocklistQueryOffsetMin)
    .exactOptional()
    .describe("How many of the newest to skip"),
  q: zod
    .string()
    .max(listBlocklistQueryQMax)
    .exactOptional()
    .describe("Only entries whose title holds this text"),
});

export const ListBlocklistResponse = GrabsBlocklist;

/**
 * @summary Remove Releases from the Blocklist by info hash. Removing one that isn't listed changes nothing
 */
export const RemoveFromBlocklistBody = UnblockInputBody;

export const RemoveFromBlocklistResponse = zod.void();

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
 * @summary Cancel a Grab that is fetching, downloading or transferring. Canceling a canceled Grab changes nothing
 */
export const CancelGrabParams = zod.object({
  grabId: zod.uuid(),
});

export const CancelGrabBody = CancelInputBody;

export const CancelGrabResponse = zod.void();

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
 * @summary Get a Work's Manual search with its plan worked out around the pins
 */
export const GetManualSearchParams = zod.object({
  workId: zod.uuid(),
});

export const getManualSearchQueryPinsMax = 2000;

export const GetManualSearchQueryParams = zod.object({
  pins: zod
    .array(zod.string())
    .max(getManualSearchQueryPinsMax)
    .nullish()
    .describe(
      "The pins from the last response, with any swap applied. A pin that no longer applies is dropped.",
    ),
});

export const GetManualSearchResponse = GrabsManualSearch;

/**
 * @summary Start a Manual search of a Work, or of some of its levels and Items or one cell. 200 answers the Work's Manual search when it already has this scope and query
 */
export const StartManualSearchParams = zod.object({
  workId: zod.uuid(),
});

export const StartManualSearchBody = GrabsStartManual;

export const StartManualSearchResponse = GrabsManualStarted;

/**
 * @summary Grab the picked Releases of a Work's Manual search in one go, each judged again and refused alone
 */
export const GrabFromManualSearchParams = zod.object({
  workId: zod.uuid(),
});

export const GrabFromManualSearchBody = GrabManualInputBody;

export const GrabFromManualSearchResponse = GrabsPickResults;

/**
 * @summary List every Release of a Work's Manual search with what grabbing it alone does
 */
export const ListManualSearchReleasesParams = zod.object({
  workId: zod.uuid(),
});

export const ListManualSearchReleasesResponse = GrabsOfferList;

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
