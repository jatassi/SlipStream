import * as zod from "zod";

import {
  CancelInputBody,
  GrabManualInputBody,
  GrabWaitingInputBody,
  GrabsBlocklist,
  GrabsGrab,
  GrabsGrabList,
  GrabsGrabbing,
  GrabsIndexerGrabList,
  GrabsManualSearch,
  GrabsManualStarted,
  GrabsOfferList,
  GrabsPickResults,
  GrabsQueuedCanceled,
  GrabsReleaseList,
  GrabsSearchList,
  GrabsSearchQueued,
  GrabsSearchScope,
  GrabsSearchSettings,
  GrabsSeveralBody,
  GrabsSeveralQueued,
  GrabsStartManual,
  GrabsWantedPage,
  GrabsWorkSearching,
  SearchSettingsInputBody,
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
 * @summary Cancel a Grab that is fetching, downloading or importing. Canceling a canceled Grab changes nothing
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
 * @summary Cancel every search queued in a lane
 */
export const CancelQueuedSearchesQueryParams = zod.object({
  lane: zod.enum(["bulk"]).describe("The lane to empty. Only bulk can be canceled"),
});

export const CancelQueuedSearchesResponse = GrabsQueuedCanceled;

/**
 * @summary Get the Automatic search settings
 */
export const GetSearchSettingsResponse = GrabsSearchSettings;

/**
 * @summary Change Automatic search settings by key, all or none
 */
export const UpdateSearchSettingsBody = SearchSettingsInputBody;

export const UpdateSearchSettingsResponse = GrabsSearchSettings;

/**
 * @summary List the Releases a search run saw, with what it did with each and why
 */
export const ListSearchReleasesParams = zod.object({
  searchId: zod.uuid(),
});

export const ListSearchReleasesResponse = GrabsReleaseList;

/**
 * @summary List the Works that want a Release, 100 at a time, with what the search queue is doing
 */
export const listWantedQueryContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const listWantedQuerySortDefault = `next`;
export const listWantedQueryLimitDefault = 100;
export const listWantedQueryLimitMax = 100;

export const ListWantedQueryParams = zod.object({
  contentType: zod
    .string()
    .regex(listWantedQueryContentTypeRegExp)
    .exactOptional()
    .describe("Left out, every Content Type"),
  reason: zod
    .array(zod.enum(["failed", "missing", "upgrade", "gone"]))
    .nullish()
    .describe("Left out, every reason but gone"),
  sort: zod
    .enum(["next", "last", "title", "released"])
    .default(listWantedQuerySortDefault)
    .describe(
      "next is the next Backlog search first, last the longest since the last search first",
    ),
  cursor: zod
    .string()
    .exactOptional()
    .describe("The next value of the page before; left out, the first page"),
  limit: zod.int().min(1).max(listWantedQueryLimitMax).default(listWantedQueryLimitDefault),
});

export const ListWantedResponse = GrabsWantedPage;

/**
 * @summary Queue a search of several Works, by id or by a Wanted filter, behind every search already queued
 */
export const SearchSeveralWorksBody = GrabsSeveralBody;

export const SearchSeveralWorksResponse = GrabsSeveralQueued;

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
 * @summary Queue a search of each wanted Item of a Work, or of some of its levels and Items, to grab the best release
 */
export const SearchWorkParams = zod.object({
  workId: zod.uuid(),
});

export const SearchWorkBody = GrabsSearchScope;

export const SearchWorkResponse = GrabsSearchQueued;

/**
 * @summary List a Work's search runs and their outcomes
 */
export const ListSearchesParams = zod.object({
  workId: zod.uuid(),
});

export const ListSearchesResponse = GrabsSearchList;

/**
 * @summary Get what Automatic search is doing for a Work: each search group's state, its waits and its stopped cells
 */
export const GetWorkSearchingParams = zod.object({
  workId: zod.uuid(),
});

export const GetWorkSearchingResponse = GrabsWorkSearching;

/**
 * @summary End a Work's waits for a better Release and grab the best Release each holds now
 */
export const GrabWaitingParams = zod.object({
  workId: zod.uuid(),
});

export const GrabWaitingBody = GrabWaitingInputBody;

export const GrabWaitingResponse = GrabsGrabbing;
