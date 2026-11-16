import { type QueryClient, infiniteQueryOptions } from "@tanstack/react-query";

import { getListWantedQueryKey, listWanted } from "../api/gen/query/grabs/grabs";
import type { Reason, Sort } from "./wanted-words";

/** What a Wanted page lists: one Content Type or every one, by reason, in a sort. */
export type WantedFilter = {
  readonly contentType: string | undefined;
  readonly reasons: readonly Reason[];
  readonly sort: Sort;
};

const paramsOf = ({ contentType, reasons, sort }: WantedFilter) => ({
  ...(contentType !== undefined && { contentType }),
  reason: [...reasons],
  sort,
});

// The key extends the generated one, so grabs.queue-changed refreshes every page loaded.

/** Wanted's rows for a filter, 100 a page, each page with what the queue is doing. */
export const wantedQuery = (filter: WantedFilter) =>
  infiniteQueryOptions({
    queryKey: [...getListWantedQueryKey(paramsOf(filter)), "pages"],
    queryFn: ({ pageParam, signal }) =>
      listWanted(
        { ...paramsOf(filter), ...(pageParam !== "" && { cursor: pageParam }) },
        { signal },
      ),
    initialPageParam: "",
    getNextPageParam: (page) => page.next,
  });

export const loadWanted = (queryClient: QueryClient, filter: WantedFilter) =>
  queryClient.infiniteQuery({ ...wantedQuery(filter), staleTime: "static" });
