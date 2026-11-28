import { type QueryClient, infiniteQueryOptions } from "@tanstack/react-query";
import { z } from "zod";

import type { ConversionsLaneRows, ListConversionsParams } from "../api/gen/model";
import {
  getListConversionsQueryKey,
  listConversions,
} from "../api/gen/query/conversions/conversions";
import { getListNodesSuspenseQueryOptions } from "../api/gen/query/nodes/nodes";
import { getListContentTypesSuspenseQueryOptions } from "../api/gen/query/works/works";
import { ensureQuery } from "../api/query-client";

/** The band the tab shows, in its address; left out, every band. */
export const conversionsSearch = z.object({
  band: z.enum(["attention", "converting", "next"]).optional().catch(undefined),
});

export type Filter = {
  readonly [K in "band" | "node" | "type" | "q"]?: ListConversionsParams[K] | undefined;
};

/** The list's params, with no empty member, so one filter keys one query. */
export const paramsOf = ({ band, node, type, q }: Filter): ListConversionsParams => ({
  ...(band !== undefined && { band }),
  ...(node !== undefined && { node }),
  ...(type !== undefined && { type }),
  ...(q !== undefined && q.trim() !== "" && { q: q.trim() }),
});

/** The tab's pages: the first carries the counts and the top bands, each one more of Up next. */
export const conversionPages = (params: ListConversionsParams) =>
  infiniteQueryOptions({
    queryKey: [...getListConversionsQueryKey(params), "pages"],
    queryFn: ({ pageParam, signal }) =>
      listConversions(
        { ...params, ...(pageParam !== undefined && { cursor: pageParam }) },
        { signal },
      ),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (page) => page.cursor ?? undefined,
  });

/** Up next across the loaded pages, a lane that runs on into the next page joined into one. */
export function lanesOf(
  pages: readonly { next: readonly ConversionsLaneRows[] }[],
): ConversionsLaneRows[] {
  const lanes: ConversionsLaneRows[] = [];
  for (const { lane, rows } of pages.flatMap((page) => page.next)) {
    const last = lanes.at(-1);
    if (last?.lane === lane) {
      last.rows = [...last.rows, ...rows];
    } else {
      lanes.push({ lane, rows: [...rows] });
    }
  }
  return lanes;
}

export const loadConversions = (queryClient: QueryClient, params: ListConversionsParams) =>
  Promise.all([
    queryClient.infiniteQuery({ ...conversionPages(params), staleTime: "static" }),
    ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
    ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
  ]);
