import { type QueryClient, hashKey, infiniteQueryOptions } from "@tanstack/react-query";

import type { HistoryListed, ListHistoryParams } from "../api/gen/model";
import {
  getGetHistorySummarySuspenseQueryOptions,
  getListHistoryKindsSuspenseQueryOptions,
  getListHistoryQueryKey,
  listHistory,
} from "../api/gen/query/history/history";
import { ensureQuery } from "../api/query-client";
import { sinceOf } from "./day-heading";

const pageSize = 50;

/** The key's last part, so a live event tells these pages from the Work page's ten rows. */
export const pagesTag = "pages";

// No maxPages: History pages by before, and a live event splices the newest
// into the first page instead of reading every page again.

const paramsByHash = new Map<string, ListHistoryParams>();

/** History's entries for params, newest first, 50 a page. */
export function historyPages(params: ListHistoryParams) {
  const queryKey = [...getListHistoryQueryKey(params), pagesTag] as const;
  paramsByHash.set(hashKey(queryKey), params);
  return infiniteQueryOptions({
    queryKey,
    queryFn: ({ pageParam, signal }) =>
      listHistory(
        { ...params, ...(pageParam !== undefined && { before: pageParam }), limit: pageSize },
        { signal },
      ),
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (page) => page.next ?? undefined,
  });
}

/** A fresh first page of the History list whose query hash is queryHash. */
export function firstPageOf(queryHash: string): Promise<HistoryListed> {
  const params = paramsByHash.get(queryHash);
  if (params === undefined) {
    throw new Error(`No History list has the query hash ${queryHash}`);
  }
  return listHistory({ ...params, limit: pageSize });
}

/** The opening sentence: the browser's last 7 days, today included. */
export const summaryOptions = (now: Date) =>
  getGetHistorySummarySuspenseQueryOptions({ since: sinceOf("7d", now) });

/** The groups and their kinds, which change only with the build. */
export const kindsOptions = () => getListHistoryKindsSuspenseQueryOptions();

export const loadHistory = (queryClient: QueryClient, params: ListHistoryParams, now: Date) =>
  Promise.all([
    queryClient.infiniteQuery({ ...historyPages(params), staleTime: "static" }),
    ensureQuery(queryClient, summaryOptions(now)),
    ensureQuery(queryClient, kindsOptions()),
  ]);
