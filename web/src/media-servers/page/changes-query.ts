import { type QueryClient, infiniteQueryOptions } from "@tanstack/react-query";

import {
  getListMediaServerChangesQueryKey,
  listMediaServerChanges,
} from "../../api/gen/query/mediaservers/mediaservers";

// The key extends the generated one, so media-servers.changed refreshes every page loaded.

/** A server's changes, newest first, a page of the API's at a time. */
export const changesQuery = (serverId: string) =>
  infiniteQueryOptions({
    queryKey: [...getListMediaServerChangesQueryKey(serverId), "pages"],
    queryFn: ({ pageParam, signal }) =>
      listMediaServerChanges(serverId, pageParam === "" ? {} : { cursor: pageParam }, { signal }),
    initialPageParam: "",
    getNextPageParam: (page) => page.next,
  });

export const loadChanges = (queryClient: QueryClient, serverId: string) =>
  queryClient.infiniteQuery({ ...changesQuery(serverId), staleTime: "static" });
