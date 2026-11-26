import { infiniteQueryOptions } from "@tanstack/react-query";

import {
  getListMediaServerPlaysQueryKey,
  listMediaServerPlays,
} from "../../api/gen/query/mediaservers/mediaservers";

// The key extends the generated one, so media-servers.plays-changed refreshes every page loaded.

/** One player's plays on a server, newest first, a page of the API's at a time. */
export const playsQuery = (serverId: string, player: string) =>
  infiniteQueryOptions({
    queryKey: [...getListMediaServerPlaysQueryKey(serverId, { player }), "pages"],
    queryFn: ({ pageParam, signal }) =>
      listMediaServerPlays(
        serverId,
        pageParam === "" ? { player } : { player, cursor: pageParam },
        { signal },
      ),
    initialPageParam: "",
    getNextPageParam: (page) => page.next,
  });
