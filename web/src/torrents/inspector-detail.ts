import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";

import {
  getGetTorrentQueryKey,
  getTorrent,
} from "../api/gen/query/downloadclients/downloadclients";
import { type LiveRow, useLiveRow } from "../live/torrents";
import { refresh } from "../live/refresh";
import { useNow } from "../tasks/use-now";

const wordsTick = 60_000;

// The Node's word on files and trackers, read again while the torrent's
// files fill, so file progress follows the live stream.
function useDetail(id: string, live: LiveRow | undefined) {
  const queryClient = useQueryClient();
  const queryKey = getGetTorrentQueryKey(id);
  const query = useQuery({ queryKey, queryFn: ({ signal }) => getTorrent(id, { signal }) });
  const filling = live !== undefined && live.done < live.wanted;
  useEffect(() => {
    if (filling) {
      void refresh(queryClient, { queryKey, exact: true });
    }
    // oxlint-disable-next-line react-hooks/exhaustive-deps -- each live sample of this torrent asks again
  }, [live]);
  return query;
}

export function useInspected(id: string) {
  const live = useLiveRow(id);
  const now = useNow(wordsTick);
  const detail = useDetail(id, live);
  const announceAt = detail.data?.live === true ? detail.data.announceAt : undefined;
  const trackersAccept = announceAt !== undefined && Date.parse(announceAt) <= now.getTime();
  return { live, clock: { now, zone: undefined }, detail, trackersAccept };
}

export type Inspected = ReturnType<typeof useInspected>;
